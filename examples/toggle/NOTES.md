# toggle

Ported from `references/xstate/examples/toggle/src/toggleMachine.ts`.

## Not ported
- `src/main.ts`: DOM rendering (button click listener, `<output>` text derived from `snapshot.value`).
- `index.html`, `src/style.css`, `src/vite-env.d.ts`, `public/`, `package.json`, `tsconfig.json`, `pnpm-lock.yaml`: markup, styling and bundler setup.

## Trace coverage (`testdata/toggle.golden.json`, recorded by `scripts/trace/toggle/toggle.ts`)
- initial state `inactive`, `toggle` to `active`, `toggle` back to `inactive`, `toggle` to `active` again.
- an unknown event while `active` (no change), then `toggle` to `inactive`.
- Both states and both transitions are visited. The machine has no context, guards, delays or invokes, so there is nothing else to cover.
