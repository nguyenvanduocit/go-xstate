# 7guis-flight-booker-react

Ported from `references/xstate/examples/7guis-flight-booker-react/src/machines/flightMachine.ts` (machine, `Booker`
actor, guards, actions) and `src/utils/index.ts` (`TODAY`, `TOMORROW`; the `sleep` helper is folded into `Booker`).

Go package name is `sevenguisflightbookerreact`: the directory name starts with a digit, which is not a valid Go identifier.

Go API differences from the JS module:
- `Machine(today, tomorrow string)` takes the dates that JS reads from the module-level `TODAY` / `TOMORROW`
  constants (computed from `new Date()` at import time). `Today()` / `Tomorrow()` compute them from the clock.
- `assertEvent` in the `setDepartDate` / `setReturnDate` actions is the type assertion `a.Event.(xs.E)["value"].(string)`.

## Not ported
- `src/App.tsx`, `src/main.tsx`, `src/components/*` (`BookButton`, `DateInput`, `Header`, `TripSelector`): React components.
- `createActorContext(flightBookerMachine)` (default export of `flightMachine.ts`): `@xstate/react` binding.
- `src/styles/*.css`, `index.html`, `public/*`: styling, HTML shell, images.
- `vite.config.ts`, `tsconfig*.json`, `package.json`, lockfiles, `.eslintrc.cjs`, `types.d.ts`, `src/vite-env.d.ts`: bundler, TypeScript and lint setup.
- The validity flags computed in `App.tsx` (`isValidDepartDate`, `isValidReturnDate = returnDate >= departDate`) are view logic, not machine logic.

## Trace coverage
Recorded by `scripts/trace/7guis-flight-booker-react/*.ts`. `harness.mjs` loads the unmodified JS machine with
`@xstate/react` replaced by an inert virtual module (React is not installed in the trace runner) and `new Date()` pinned
to 2024-03-10T12:00:00Z while the module loads, so TODAY = `2024-03-10` and TOMORROW = `2024-03-11`. The Go test uses the
same dates. The real `Booker` sleeps 2000 ms; both sides swap in a stub that settles after 100 ms of real time (`{wait: 300}`
steps let it settle), so the machine stays in `booking` while the following steps run.

`testdata/flight-booker.golden.json` (`flight-booker.ts`, stub resolves):
- initial `scheduling.oneWay` with context `{departDate: TODAY, returnDate: TOMORROW}`.
- `CHANGE_RETURN_DATE` and `BOOK_RETURN` ignored in `oneWay`; `BOOK_DEPART` ignored in `roundTrip`.
- `CHANGE_DEPART_DATE` (handled on `scheduling`) in both sub-states; `CHANGE_RETURN_DATE` in `roundTrip`.
- `isValidDepartDate?` false (depart date before today; `BOOK_DEPART` stays in `oneWay`).
- `isValidReturnDate?` false three ways: return < depart, return == depart, depart before today with return > depart.
- `CHANGE_TRIP_TYPE` in both directions (`oneWay` -> `roundTrip` -> `oneWay` -> `roundTrip`).
- `isValidReturnDate?` true: `scheduling.roundTrip` -> `booking`; `CHANGE_DEPART_DATE` ignored in `booking`.
- invoke `onDone`: `booking` -> `booked` (final, snapshot status `done`, children empty); event after final ignored.

`testdata/flight-booker-oneway-error.golden.json` (`flight-booker-oneway-error.ts`, stub rejects):
- `isValidDepartDate?` true: `scheduling.oneWay` -> `booking`; `CHANGE_DEPART_DATE` / `CHANGE_TRIP_TYPE` ignored in `booking`.
- invoke `onError`: `booking` -> `scheduling` (re-enters `oneWay`, context unchanged), then a second booking and error.

Not covered by a golden trace: the real 2000 ms `Booker` (a trace would take 2 s per booking); `TestBookerStopsOnCancel`
checks that it is invoked and that stopping the actor cancels its context promptly. `Today()` / `Tomorrow()` depend on the
wall clock and are only format-checked (`TestDates`).
