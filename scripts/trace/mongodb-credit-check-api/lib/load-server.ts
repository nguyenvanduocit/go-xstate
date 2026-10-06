// Runs the JS example's own index.ts (real handlers, real actorService.ts, real machine.ts)
// against an in-process router and an in-memory `mongodb`, with the stub services of
// lib/stub-services.ts. `express`, `body-parser` and `mongodb` are replaced by fakes because
// they are not installed. Math.random is a fixed sequence, so workflow ids are deterministic.
import { plugin } from 'bun';
import './load.ts';

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
  const addRoute = (method: 'GET' | 'POST') => (path: string, fn: Handler) => {
    const keys: string[] = [];
    const src = path.replace(/:(\w+)/g, (_, k) => {
      keys.push(k);
      return '([^/]+)';
    });
    routes.push({ method, pattern: new RegExp(`^${src}$`), keys, fn });
  };
  const fakeApp = { use() {}, get: addRoute('GET'), post: addRoute('POST'), listen(_port: number, cb: () => void) { cb(); } };

  // In-memory collection: documents are stored as JSON, like BSON drops `undefined`.
  const docs = new Map<string, unknown>();
  const collection = () => ({
    async findOne(filter: { workflowId: string }) {
      const d = docs.get(filter.workflowId);
      return d === undefined ? null : JSON.parse(JSON.stringify(d));
    },
    async replaceOne(filter: { workflowId: string }, doc: unknown) {
      docs.set(filter.workflowId, JSON.parse(JSON.stringify(doc)));
      return { acknowledged: true };
    }
  });
  class MongoClient {
    constructor(_uri: string, _opts: unknown) {}
    db(_name: string) {
      return { collection };
    }
    async connect() {}
  }

  plugin({
    name: 'fake-express-mongodb',
    setup(build) {
      build.module('express', () => ({ exports: { default: () => fakeApp }, loader: 'object' }));
      build.module('body-parser', () => ({ exports: { default: { json: () => () => {} } }, loader: 'object' }));
      build.module('mongodb', () => ({
        exports: { MongoClient, ServerApiVersion: { v1: '1' }, ObjectId: class {} },
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

  await import('../../../../references/xstate/examples/mongodb-credit-check-api/index.ts');
  await new Promise((r) => setTimeout(r, 0)); // initDbConnection().then(listen)

  async function request(method: 'GET' | 'POST', path: string, body?: unknown): Promise<Response> {
    for (const r of routes) {
      if (r.method !== method) continue;
      const m = r.pattern.exec(path);
      if (!m) continue;
      const params: Record<string, string> = {};
      r.keys.forEach((k, i) => (params[k] = m[i + 1]!));
      const out: Response = { status: 200 };
      // Express throws ERR_HTTP_HEADERS_SENT on a second send; the first response is what the client sees.
      let sent = false;
      const res: any = {
        status(code: number) {
          if (!sent) out.status = code;
          return res;
        },
        send(x: unknown) {
          if (sent) return res;
          sent = true;
          if (typeof x === 'string') out.text = x;
          else out.json = JSON.parse(JSON.stringify(x));
          return res;
        },
        json(x: unknown) {
          if (sent) return res;
          sent = true;
          out.json = JSON.parse(JSON.stringify(x));
          return res;
        }
      };
      await r.fn({ params, body: body ?? {} }, res);
      return out;
    }
    return { status: 404, text: 'Cannot ' + method + ' ' + path };
  }

  return { request, restoreRandom: () => (Math.random = realRandom) };
}
