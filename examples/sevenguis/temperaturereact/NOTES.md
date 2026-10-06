# 7guis-temperature-react

Ported from `references/xstate/examples/7guis-temperature-react/src/temperatureMachine.ts`.

## Not ported
- `src/App.tsx`, `src/main.tsx`: React components and `useMachine` wiring (UI rendering).
- `src/App.css`, `src/index.css`, `src/favicon.svg`, `index.html`: styling and HTML.
- `vite.config.ts`, `tsconfig.json`, `package.json`, `pnpm-lock.yaml`, `readme.md`: bundler and package config.
- JS number semantics are ported only for what a string input can produce: `toNumber` in `machine.go` implements unary `+` on strings (trim, empty, decimal literals, `0x`/`0o`/`0b`, `Infinity`, otherwise NaN). `Number` serialises NaN and +-Infinity as `null`, like `JSON.stringify`.

## Trace coverage (`testdata/temperature.golden.json`, recorded by `scripts/trace/7guis-temperature-react/temperature.ts`)
- The machine has no states, guards, delays or invokes; the root has two events, each with one branch on `value.length`.
- `CELSIUS`: non-empty value (`100`, `-40`, `36.6`, `1e2`, padded `" 12 "`, hex `0x10`, whitespace-only `" "`), empty value, NaN (`abc`), Infinity.
- `FAHRENHEIT`: non-empty value (`212`, `-40`, `98.6`, `.5`), empty value, NaN (`abc`), `-Infinity`.
- initial context (`{}`, both fields undefined) and an unknown event (no change).
