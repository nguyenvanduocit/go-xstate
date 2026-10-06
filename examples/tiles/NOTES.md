# tiles

Ported from `references/xstate/examples/tiles/src/tilesMachine.ts` (`machine.go`).

## Not ported
- `src/App.tsx`, `src/main.tsx`, `src/App.css`, `src/index.css`, `index.html`, `public/`, Vite/TS config, `package.json`: React rendering (tile grid, highlight styling, click/hover handlers), CSS and bundler setup.
- Go-side differences that are not behaviour changes:
  - `swap` returns a copy instead of mutating `context.tiles` in place (the JS `swap` mutates the array; the snapshots are identical).
  - `Math.random` is injected: `NewMachine(random)`; `Machine()` uses `math/rand`.
  - Absent `selected`/`hovered` (JS `undefined`) are nil pointers with `omitempty`, so the key is absent from the JSON snapshot as in JS.

## Trace coverage (`testdata/tiles.golden.json`, recorded by `scripts/trace/tiles/tiles.ts`)
- Math.random is replaced in the JS script by a fixed sequence (an exhausted sequence throws): SWAP01 (14 x 0.99 then 0: Fisher-Yates swaps tiles 0 and 1 only), IDENTITY (15 x 0.99), SWAP01. The Go test feeds the same sequence.
- `start`: ignores `unknown`, `tile.select`, `tile.move`, `move.canceled`, `tile.hover`; root `shuffle` goes to `playing.selecting` with a non-ordered board (`always` guard false).
- `playing.selecting`: `shuffle` swallowed by `playing`'s targetless transition (no random consumed); `tile.hover`, `tile.move`, `move.canceled` have no handler; `tile.select` -> `selected`.
- `playing.selected`: `shuffle` swallowed; `tile.select` ignored; `tile.hover` (repeated); `move.canceled` clears selected and hovered; `tile.move` with every `isAdjacent` branch: hovered undefined, diagonal, same row two apart, same column two apart (all false, board unchanged, selected kept), horizontal adjacent, vertical adjacent (true, swap + clear both).
- `allTilesInOrder`: false after moves, true after the final swap -> `gameOver`; also true right after a `shuffle` from `gameOver` with IDENTITY (playing entered, back to `gameOver` in the same macrostep).
- `gameOver`: ignores `tile.select`, `tile.move`, `move.canceled`; `shuffle` -> `playing.selecting`.
- Not in the golden trace: the production `math/rand` shuffle (covered by `TestMachineWithMathRandom` as a permutation check only).
