# tic-tac-toe-react

Ported from `references/xstate/examples/tic-tac-toe-react/src/ticTacToeMachine.ts`.

## Not ported
- `src/App.tsx`, `src/main.tsx`, `src/styles.css`, `src/vite-env.d.ts`: React rendering (`useMachine`, `Tile`, `range`), CSS, DOM mount.
  The UI only reads `state.matches('gameOver')`, `hasTag('winner'|'draw')`, `context.winner` and `context.board`, all of which the golden snapshots record.
- `index.html`, `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `readme.md`: bundler and package setup.
- `assertEvent` (TypeScript narrowing helper): replaced by an inline `panic("Unexpected event type.")` in `updateBoard`; unreachable because only `PLAY` reaches it.

## Go representation notes
- `null` board cells and `undefined` winner map to the empty `Player`: `MarshalJSON` emits `null`, and `winner` carries `omitempty`, so the JSON equals the JS snapshot (`winner` key absent until set and after `resetGame`).
- `isValidMove` treats a non-integer, negative, too large or non-numeric `value` as invalid (JS `board[value] === null` is false for those). Covered by the `9` and `-1` steps.
- `PLAY` values are `int` when sent from Go (`Play(i)`) and `float64` when replayed from a golden file; `playIndex` accepts both (`TestGoEvents` covers the `int` path).

## Trace coverage (`testdata/*.golden.json`, recorded by `scripts/trace/tic-tac-toe-react/<name>.ts`)
- `x-wins`: ignored `RESET` and unknown event in `playing`; valid `PLAY`; `isValidMove` false (occupied cell, index 9, index -1); `checkWin` true (top row) -> `gameOver.winner` with tag `winner` and `setWinner` (winner `x`); `PLAY` ignored in `gameOver`; `RESET` -> `playing` with `resetGame`; `PLAY` on the fresh board.
- `o-wins`: `checkWin` true for `o` (middle row) -> winner `o`; `RESET`.
- `draw`: `checkWin` false and `checkDraw` true after move 9 -> `gameOver.draw` with tag `draw`; `PLAY` ignored in `gameOver.draw`; `RESET`; `PLAY`.
- `win-on-last-move`: ninth move completes a line, so the `always` order puts `checkWin` before `checkDraw` (`gameOver.winner`, not `draw`).
- All states visited: `playing`, `gameOver.winner`, `gameOver.draw`. Both branches of `checkWin`, `checkDraw` and `isValidMove`. The machine has no delays, invokes or `onDone`.
- `TestGoEvents` (Go only): `Play(int)` events and snapshot immutability of `Board` across moves.
