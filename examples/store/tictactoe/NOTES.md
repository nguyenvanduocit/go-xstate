# store-tic-tac-toe

Ported from `references/xstate/examples/store-tic-tac-toe/src/store.ts` (the whole file: `getGameOutcome` and `gameStore`) and the
store reads of `src/App.tsx` (`useSelector(gameStore, ...)` for `context.board`, `context.currentPlayer`, `context.status`).

Go: `Config()` is the `createStore` config (`store.ts:51-82`), `GetGameOutcome` is `getGameOutcome` (`store.ts:25-49`), `NewGame()` is
`createStore(config)` plus the three selectors (`xstore.Select`). `Context.Board` is a `[9]Mark` value array; the empty `Mark` is JS
`null` and marshals to JSON `null`, so Go snapshots compare equal to the JS ones.

## Behaviour carried over from the JS source
- A draw ends with `context.status == "won"`: `store.ts:68-72` tests `outcome.winner ? 'won' : outcome.winner === 'draw' ? 'draw' : 'playing'`,
  and `'draw'` is truthy, so the `'draw'` branch never runs. The Go handler reproduces this; golden step 28 records it
  (`outcome.winner: "draw"`, `context.status: "won"`). The UI still shows "Draw!" because `App.tsx` derives it from `getGameOutcome`.
- A `played` event with a position outside 0-8 (also fractional or missing) is ignored: JS `board[9]` is `undefined`, never `null`.

## Not ported
- `src/App.tsx` JSX and the render-time derivations inside it (`Square` variant `winning`/`played`/`unplayed`, the "Winner: x" /
  "Draw!" / "Current player: x" text, showing the "Play Again" button when `status !== 'playing'`): React rendering; they are inline in
  components and cannot be imported by the trace runner.
- `@xstate/store-react` (`useSelector`): the React hook binding; the Go `store` package is the `@xstate/store` part.
- `src/main.tsx`, `src/App.css`, `src/index.css`, `index.html`, `public/vite.svg`, `src/vite-env.d.ts`, `vite.config.ts`,
  `eslint.config.js`, `tsconfig*.json`, `package.json`, `README.md`: bundler, lint, styling and HTML.

## Trace coverage
JS scripts live in `scripts/trace/store-tic-tac-toe/`. The JS store module is loaded unmodified from the example directory by
`lib/load.ts`: the example's own `package.json` (`@xstate/store ^3.17.1`) stops Bun from resolving `@xstate/store` from there and the
`preload.ts` mapping does not apply, so `lib/load.ts` rewrites that one import specifier at load time to `packages/xstate-store/src`.

- `testdata/store.golden.json` (`store.ts`, steps in `lib/steps.ts`): `gameStore` + `store.inspect(recorder)` + the three selectors
  subscribed. Per step it compares `status`, `context`, `getGameOutcome(board)`, the inspection events received (init event on `inspect`,
  one `@xstate.transition` per `send`, including ignored and unknown events) and the values each selector subscription was notified with.
  Covers: a move on an empty cell (both players), a move on a taken cell, positions 9 and -1, a win for x (top row), a move after the game
  ended, `reset` from a finished game, `reset` on an initial store, a win for o (middle row), an event type with no handler, a full board
  with no line (draw, status `won`), a move after a draw, `reset` after a draw.
- `testdata/outcome.golden.json` (`outcome.ts`): `getGameOutcome` over the 8 lines for x, a win for o, two complete lines (the first in
  `lines` order wins), a draw, an empty board and a board in play.
- `TestGoCallerPayload` (Go only): `Trigger`/`Send` with `int` positions and the fractional/missing position cases; JS numbers are untyped.
