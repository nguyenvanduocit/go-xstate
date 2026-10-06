# counter

Ported from `references/xstate/examples/counter/src/counterMachine.ts`.

## Not ported
- `src/main.ts`, `index.html`, `public/`, Vite config: DOM rendering and bundler setup.

## Trace coverage (`testdata/counter.golden.json`, recorded by `scripts/trace/counter/counter.ts`)
- initial context, `increment` x2, `decrement`, and an unknown event (no change).
- The machine has no states, guards, delays or invokes, so there is nothing else to cover.
