# snake-react

Ported from `references/xstate/examples/snake-react/src/snakeMachine.ts` (machine, `getGamObjectAtPos` and the
pure helpers) to `machine.go`.

## Not ported
- `src/App.tsx`, `src/main.tsx`, `src/index.css`, `src/vite-env.d.ts`, `index.html`: React rendering (the keyboard
  handler, the grid drawing and `useMachine`), styling and bundler entry.
- `package.json`, `vite.config.ts`, `tsconfig*.json`, `.eslintrc.cjs`, `pnpm-lock.yaml`, `README.md`: bundler,
  lint and package setup.
- `Math.random` and `setInterval(80)` are injected rather than hard-coded: `NewMachine(random, tickInterval)`;
  `Machine()` uses `math/rand/v2` and `TickInterval` (80 ms). The JS script pins both for the trace (below).

## Traces
`testdata/snake.golden.json`, recorded by `scripts/trace/snake-react/snake.ts`. `Math.random` returns a fixed
sequence and the `ticks` actor is replaced (`machine.provide`) by a no-op callback; the script sends `TICK` itself.
The Go test uses the same random sequence and a no-op `ticks` stub. Covers:
- `New Game`: `TICK` and `NEW_GAME` ignored; `ARROW_KEY` -> `Moving` (entry `move snake`, invoke `ticks` visible as
  child `0.SnakeMachine.Moving`).
- `Moving`: `TICK` moves; `ARROW_KEY` keeps the state without re-entering (it only saves `dir`; the snake moves on the
  next `TICK`); `NEW_GAME` ignored.
- `always` guards: `ate apple` true (twice; `show new apple` retries once when the first random point lies on the
  snake) and false; `hit tail` true; `hit wall` true with `hit tail` false (both sides of `or`).
- `save dir`: new direction accepted; opposite direction ignored (in `Moving` and in `New Game`).
- `increase score`: `highScore` raised (score+1 > highScore) and kept (score+1 < highScore, second game).
- `Game Over`: `TICK` and `ARROW_KEY` ignored; `NEW_GAME` -> `New Game` via `reset`, which keeps `highScore`.

`testdata/gameobjects.golden.json`, recorded by `scripts/trace/snake-react/gameobjects.ts`: `getGamObjectAtPos`
for every cell of the 25x15 grid plus a one-cell border, for the initial context and a mid-game context
(head, body with differing dirs, apple, empty). Replayed by `TestGameObjectAtPos`.

`TestRealTicks` runs the real `ticks` actor (5 ms interval) with no scripted `TICK`; it has no JS golden (the JS
actor uses real `setInterval`, which is not deterministic) and checks only that ticks arrive until `Game Over`
and that the actor stops with the state.
