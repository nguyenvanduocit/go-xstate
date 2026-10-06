# local-store-counter-react

Ported from `references/xstate/examples/local-store-counter-react/src/App.tsx` (the `useStore` config inside `Counter`, lines 5-20,
and the `useSelector(store, (s) => s.context.count)` selector, line 21).

Go: `Config(initialCount)` is the `useStore` config; `NewCounter(initialCount)` is `createStore(config)` plus the count selector
(`xstore.Select`); `InitialCounts` = the three `initialCount` props `App` renders (0, 10, 100).

## Not ported
- `src/App.tsx` JSX (the two `<button>`s, `className="card"`, inline flex style, `onClick` wiring) and `src/main.tsx`: React rendering.
- `useStore` / `useSelector` from `@xstate/store-react`: the React hook binding; the Go `store` package is the `@xstate/store` part.
- `src/App.css`, `src/index.css`, `src/assets/react.svg`, `index.html`, `vite.config.ts`, `eslint.config.js`, tsconfig files, `pnpm-lock.yaml`, `README.md`: styling and bundler/lint setup.
- `@statelyai/inspect` dependency: not imported by any source file of the app.

## Trace coverage
The JS trace scripts live in `scripts/trace/local-store-counter-react/`. `App.tsx` cannot be imported by the trace runner
(React, CSS and `@xstate/store-react` are not resolvable), so `lib/counterConfig.ts` repeats the config of `App.tsx:5-20` verbatim.

Every trace runs the same steps: `inc by 1`, `inc by 5`, `reset`, `inc by 1` (after reset), `inc by -3` (negative result), unknown
event type (no handler, no change), `reset`. This takes both `on` handlers and the no-handler path.

- `testdata/counter-0|10|100.golden.json` (`counter-<n>.ts`): the config as a `fromStore` actor replayed with `tracetest.Run`
  (status, context, output, tags, children after every step).
- `testdata/store.golden.json` (`store.ts`): the path `App.tsx` actually uses, `createStore(config)` + `store.select(count)`, for
  the three initial counts; compares status, context and selected count.
- `TestGoCallerPayload` (Go only): `Trigger`/`Send` with `int` payloads and selector subscription notifications; not in a golden
  because JS numbers are untyped.
