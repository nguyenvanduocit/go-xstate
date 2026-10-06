# 7guis-1-counter-vue

Ported from `references/xstate/examples/7guis-1-counter-vue/src/counterMachine.ts`.

## Not ported
- `src/Counter.vue`, `src/App.vue`, `src/components/HelloWorld.vue`: Vue components using `@xstate/vue` `useMachine` (DOM rendering).
- `src/main.ts`, `src/style.css`, `src/vue-shim.d.ts`, `src/vite-env.d.ts`, `index.html`, `vite.config.ts`, `tsconfig*.json`: Vue bootstrap, CSS, type shims and bundler config.

## Trace coverage (`testdata/counter.golden.json`, recorded by `scripts/trace/7guis-1-counter-vue/counter.ts`)
- initial state `ready` with count 0, `increase` x3 (self-transition `ready` -> `ready` with assign), and an unknown event (no change).
- The machine has one state and no guards, delays or invokes, so nothing else exists to cover.
