# store-counter-react

Ported from `references/xstate/examples/store-counter-react/src/App.tsx`: the module-level `createStore` call (lines 6-21), the
`store.inspect(...)` call (line 23) and the `useSelector(store, (s) => s.context.count)` selector (line 26).

Go: `Config()` is the `createStore` config; `NewCounter()` is `createStore(config)` plus the count selector (`xstore.Select`).
`Counter.Store.Inspect` is the Go form of `store.inspect`.

## Not ported
- `src/App.tsx` JSX (the two `<button>`s, `className="card"`, inline flex style, `onClick` wiring), `src/main.tsx`: React rendering.
- `useSelector` from `@xstate/store-react`: the React hook binding; the Go `store` package is the `@xstate/store` part.
- `createBrowserInspector()` from `@statelyai/inspect` (App.tsx lines 4-6, 23): opens a browser inspector window. The Go port keeps
  the `store.inspect(fn)` call itself (tests pass a recorder); the browser inspector sink has no Go equivalent.
- `src/App.css`, `src/index.css`, `src/assets/react.svg`, `index.html`, `vite.config.ts`, `eslint.config.js`, tsconfig files,
  `pnpm-lock.yaml`, `README.md`: styling and bundler/lint setup.

## Trace coverage
The JS trace scripts live in `scripts/trace/store-counter-react/`. `App.tsx` cannot be imported by the trace runner (React,
CSS, `@xstate/store-react` and `@statelyai/inspect` are not resolvable), so `lib/counterStore.ts` repeats the config of
`App.tsx:6-21` verbatim.

Both traces run the same steps: `inc by 1` x2, `inc by 5`, `reset`, `inc by 1` (after reset), `inc by -3` (negative result),
unknown event type (no handler, no change), `reset`, `reset` (already 0). This takes both `on` handlers and the no-handler path.

- `testdata/counter.golden.json` (`counter.ts`): the config as a `fromStore` actor replayed with `tracetest.Run` (status, context,
  output, tags, children after every step).
- `testdata/store.golden.json` (`store.ts`): the path `App.tsx` uses, `createStore(config)` + `store.inspect(recorder)` +
  `store.select(count)`. Per step it compares status, context, selected count, the inspection events received (init event on
  `inspect`, one `@xstate.transition` per `send`, including the unknown event) and the values the selector subscription was
  notified with (no notification when the count is unchanged: unknown event, second `reset`).
- `TestGoCallerPayload` (Go only): `Trigger`/`Send` with `int` payloads; not in a golden because JS numbers are untyped.
