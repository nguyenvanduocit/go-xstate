# fetch

Ported from `references/xstate/examples/fetch/src/fetchMachine.ts` and the non-UI parts of `src/index.ts`
(`getGreeting`, the console entry). Go: `machine.go` (`NewMachine`, `Machine`, `GetGreeting`, `Run`, `RunWith`).

## Not ported
- `index.html`, `package.json` (Parcel bundler/scripts), `pnpm-lock.yaml`, `README.md`: HTML shell and bundler setup.
- `getGreeting` hardcodes `setTimeout(1000)` and `Math.random()`; Go takes `delay` and `random` as parameters
  (`Machine()` passes 1 s and `math/rand/v2.Float64`). The JS `rej()` (reject with `undefined`) becomes `ErrRejected`.
- The JS entry's `console.log` output uses the runtime's object formatter; `formatContext` reproduces that layout only for this
  context shape (bun style: multi-line, double quotes, trailing commas).
- Import-order quirk: `fetchMachine.ts` and `index.ts` import each other and `index.ts` starts a demo actor on import. The
  JS trace script (`scripts/trace/fetch/fetch.ts`) loads `index.ts` first with `console.log` muted and `Math.random`
  pinned, then swaps `fetchUser` through `machine.provide`. Go has no import cycle or import-time side effect.

## Trace coverage
`testdata/fetch.golden.json`, recorded by `scripts/trace/fetch/fetch.ts` (SimulatedClock, `fetchUser` stubbed: attempts 1 and 2
reject, attempt 3 resolves):
- every state: `idle`, `loading`, `success`, `failure`.
- `FETCH` (idle -> loading); `onError` -> failure (twice); `onDone` -> success with `data` assigned from the output; input `{name}` taken from context.
- `RETRY` in failure -> loading; `after: 1000` in failure -> loading, checked at 999 ms (still failure) and 1000 ms (fires).
- the `after` timer of the first failure is cancelled when `RETRY` leaves the state (it would fire during the later 500 ms advance).
- ignored events: `RETRY` in idle, `FETCH` in failure and success, `RETRY` in success; no timers in idle or success.
- the invoked child appears in `children` while loading (`0.(machine).loading`) and is gone afterwards.

`testdata/fetch.stdout.txt`, recorded from `src/index.ts` with `Math.random` pinned to 0.9 (resolves): idle, loading, success. The Go test
runs `RunWith` with a 10 ms delay. The failure branch of `getGreeting` is unit-tested in `TestGetGreeting` (no JS golden: the entry
cannot print it deterministically without changing it).

`GetGreeting` honours context cancellation (Go idiom for abandoning the promise); the JS promise keeps running when the state exits.
