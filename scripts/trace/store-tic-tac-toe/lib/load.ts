// Loads the JS example's own store module, unmodified on disk. The example
// directory has its own package.json (`@xstate/store ^3.17.1`), so Bun cannot
// resolve '@xstate/store' from inside it and the preload.ts mapping does not
// apply there. This runtime plugin rewrites that one import specifier, while
// the file is loaded, to the @xstate/store sources of the repo: the target
// preload.ts maps it to for scripts under scripts/trace.
import { plugin } from 'bun';
import { resolve } from 'node:path';

const storeSrc = resolve(import.meta.dir, '../../../../references/xstate/packages/xstate-store/src/index.ts');
const exampleStore = resolve(import.meta.dir, '../../../../references/xstate/examples/store-tic-tac-toe/src/store.ts');

plugin({
  name: 'store-tic-tac-toe-xstate-store',
  setup(build) {
    build.onLoad({ filter: /store-tic-tac-toe\/src\/store\.ts$/ }, async (args) => {
      const source = await Bun.file(args.path).text();
      return {
        contents: source.replaceAll("'@xstate/store'", JSON.stringify(storeSrc)),
        loader: 'ts'
      };
    });
  }
});

export const { gameStore, getGameOutcome } = await import(exampleStore);
