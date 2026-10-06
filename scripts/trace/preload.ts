// Resolve `xstate` and `@xstate/store` to the TypeScript sources in the cloned repo,
// so example code runs against the exact reference implementation without a build.
import { plugin } from 'bun';
import { resolve } from 'node:path';

const root = resolve(import.meta.dir, '../../references/xstate/packages');
const map: Record<string, string> = {
  xstate: `${root}/core/src/index.ts`,
  'xstate/graph': `${root}/core/src/graph/index.ts`,
  '@xstate/store': `${root}/xstate-store/src/index.ts`
};

plugin({
  name: 'xstate-src',
  setup(build) {
    build.onResolve({ filter: /^(xstate(\/graph)?|@xstate\/store)$/ }, (args) => ({
      path: map[args.path]
    }));
  }
});
