// Loads the JS example's own index.ts (references/xstate/examples/express-workflow/index.ts)
// with `express` and `body-parser` replaced by a minimal in-process router, so the real
// handlers run without a network or the (not installed) express package.
// Math.random is replaced by a fixed sequence, so generated workflow ids are deterministic.
import { plugin } from 'bun';

type Handler = (req: any, res: any) => unknown;
interface Route {
  method: 'GET' | 'POST';
  pattern: RegExp;
  keys: string[];
  fn: Handler;
}

export interface Response {
  status: number;
  json?: unknown;
  text?: string;
}

export async function loadServer(randoms: number[]) {
  const routes: Route[] = [];
  let listenCb: (() => void) | undefined;
  const addRoute = (method: 'GET' | 'POST') => (path: string, fn: Handler) => {
    const keys: string[] = [];
    const src = path.replace(/:(\w+)/g, (_, k) => {
      keys.push(k);
      return '([^/]+)';
    });
    routes.push({ method, pattern: new RegExp(`^${src}$`), keys, fn });
  };
  const fakeApp = {
    use() {},
    get: addRoute('GET'),
    post: addRoute('POST'),
    listen(_port: number, cb: () => void) {
      listenCb = cb;
    }
  };
  plugin({
    name: 'fake-express',
    setup(build) {
      build.module('express', () => ({
        exports: { default: () => fakeApp },
        loader: 'object'
      }));
      build.module('body-parser', () => ({
        exports: { default: { json: () => () => {} } },
        loader: 'object'
      }));
    }
  });

  const queue = [...randoms];
  const realRandom = Math.random;
  Math.random = () => {
    const v = queue.shift();
    if (v === undefined) throw new Error('Math.random sequence exhausted');
    return v;
  };

  const realLog = console.log;
  const printed: string[] = [];
  console.log = (...a: unknown[]) => void printed.push(a.join(' '));
  try {
    await import('../../../../references/xstate/examples/express-workflow/index.ts');
    listenCb?.();
  } finally {
    console.log = realLog;
  }

  function request(method: 'GET' | 'POST', path: string, body?: unknown): Response {
    for (const r of routes) {
      if (r.method !== method) continue;
      const m = r.pattern.exec(path);
      if (!m) continue;
      const params: Record<string, string> = {};
      r.keys.forEach((k, i) => (params[k] = m[i + 1]!));
      const out: Response = { status: 200 };
      const res: any = {
        status(code: number) {
          out.status = code;
          return res;
        },
        send(x: unknown) {
          if (typeof x === 'string') out.text = x;
          else out.json = JSON.parse(JSON.stringify(x));
          return res;
        },
        json(x: unknown) {
          out.json = JSON.parse(JSON.stringify(x));
          return res;
        },
        sendStatus(code: number) {
          out.status = code;
          out.text = code === 200 ? 'OK' : String(code);
          return res;
        }
      };
      r.fn({ params, body: body ?? {} }, res);
      return out;
    }
    return { status: 404, text: 'Cannot ' + method + ' ' + path };
  }

  return { request, printed, restoreRandom: () => (Math.random = realRandom) };
}
