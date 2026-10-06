# 7guis-2-temperature-vue

Ported from `references/xstate/examples/7guis-2-temperature-vue/src/tempMachine.ts` (`machine.go`).

## Not ported
- `src/App.vue`, `src/TempConverter.vue` (Vue components, `useMachine` from `@xstate/vue`, the two text inputs and `@input` handlers), `src/main.ts`, `src/style.css`, `index.html`, `public/`, `vite.config.ts`, `tsconfig*.json`, `src/vue-shim.d.ts`, `src/vite-env.d.ts`, `pnpm-lock.yaml`: DOM rendering and bundler setup.
- JS number semantics that the machine relies on (`+event.value`, `Math.round`, `String.prototype.trim`) have no Go equivalent, so `machine.go` carries small helpers (`toNumber`, `jsRound`, `jsTrim`, `Num`) instead of using `strconv`/`math.Round` directly. `toNumber` covers decimal, `Infinity`, and `0x`/`0o`/`0b` literals and JS whitespace, which is the whole StringToNumber grammar. Radix literals above 2^53 round via `big.Float` (nearest-even, as in JS).

## Trace coverage (`testdata/temperature.golden.json`, recorded by `scripts/trace/7guis-2-temperature-vue/temperature.ts`)
- The machine has one state (`ready`), so every state is visited; no delays or invokes exist.
- `changeC` and `changeF`: guard true with non-empty value, guard true with whitespace-only value (`onChange*` clears both fields), guard false (`abc`, `1_0`, `1e`, `-0x10`: no change).
- Unary-plus literals: exponent, surrounding whitespace, tab/newline, hex/binary/octal, leading `.`, trailing `.`, leading `+`, `Infinity`/`-Infinity` (serialised as `null`), `-0` (serialised as `0`).
- `Math.round` ties: `2.5` C -> 37 (half up), `-37.5` C -> -35 and `-39.1` F -> -39 (negative half rounds toward +Infinity, unlike Go `math.Round`).
- An unknown event (no change).
