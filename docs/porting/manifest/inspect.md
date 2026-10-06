> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: inspect_1

Source: `references/xstate/packages/core/test/inspect.test.ts` lines 1-1153 (whole file, 12 tests).
Go file: `xstate/inspect_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| inspect > the .inspect option can observe inspection events | TestInspect_TheInspectOptionCanObserveInspectionEvents | ported | session ids normalized (see below) |
| inspect > can inspect communications between actors | TestInspect_CanInspectCommunicationsBetweenActors | ported | uses gaps `PromiseResolveEvent`, `WithInspectObserver`; waits for the promise's final snapshot before asserting |
| inspect > can inspect microsteps from always events | TestInspect_CanInspectMicrostepsFromAlwaysEvents | ported | raw inline snapshot asserted field by field |
| inspect > can inspect microsteps from raised events | TestInspect_CanInspectMicrostepsFromRaisedEvents | ported | raise params as `map[string]any{"delay","event","id"}` |
| inspect > should inspect microsteps for normal transitions | TestInspect_ShouldInspectMicrostepsForNormalTransitions | ported | |
| inspect > should inspect microsteps for eventless/always transitions | TestInspect_ShouldInspectMicrostepsForEventlessAlwaysTransitions | ported | |
| inspect > should inspect actions | TestInspect_ShouldInspectActions | ported | inline action type `(anonymous)` |
| inspect > @xstate.microstep inspection events should report no transitions if an unknown event was sent | TestInspect_MicrostepInspectionEventsShouldReportNoTransitionsIfAnUnknownEventWasSent | ported | `expect.assertions(1)` → assert exactly one microstep event |
| inspect > actor.system.inspect(…) can inspect actors | TestInspect_ActorSystemInspectCanInspectActors | ported | |
| inspect > actor.system.inspect(…) can inspect actors (observer) | TestInspect_ActorSystemInspectCanInspectActorsObserver | ported | uses gap `ActorSystem.InspectObserver` |
| inspect > actor.system.inspect(…) can be unsubscribed | TestInspect_ActorSystemInspectCanBeUnsubscribed | ported | |
| inspect > actor.system.inspect(…) can be unsubscribed (observer) | TestInspect_ActorSystemInspectCanBeUnsubscribedObserver | ported | uses gap `ActorSystem.InspectObserver` |

Totals: 12 JS tests, 12 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_inspect_1.go`)

- `type PromiseResolveEvent struct{ Data any }` with `EventType() == "xstate.promise.resolve"` — mirrors the
  internal promise-actor event (`actors/promise.ts` `XSTATE_PROMISE_RESOLVE`) visible in `@xstate.event`
  inspection events (JS lines 382-389, 431-436).
- `func WithInspectObserver(observer Observer[InspectionEvent]) ActorOption` — mirrors
  `createActor(logic, { inspect: { next } })` (JS lines 222-227); the contract's `WithInspect` takes only a function.
- `func (s *ActorSystem) InspectObserver(observer Observer[InspectionEvent]) Subscription` — mirrors
  `system.inspect({ next })` (JS lines 1085-1107, 1130-1151). The contract's `Inspect` takes only a function.

## Ambiguities for the reviewer

1. Session ids (JS `"x:0"` … `"x:7"`): JS uses a module-level counter (`system.ts:90`), so absolute values
   depend on test order. Go cannot reproduce that under `-run`/shuffle, so `inspect1Labeler` assigns the JS
   labels to distinct session ids in order of first appearance in the simplified stream, and each test also
   asserts the root actor maps to its JS label (e.g. `x:1` in the communications test). Distinct actors still
   must have distinct session ids, and the same actor must keep one id. Test 3 (raw snapshot, JS `x:4`) asserts
   `actorRef.SessionID()` and `rootId` equal `actor.SessionID()` instead.
2. Raise action params (JS lines 731-753): JS `{ delay: undefined, event: {type}, id: undefined }` is ported as
   `map[string]any{"delay": nil, "event": xs.Ev("to_b"), "id": nil}`. The contract types `InspectedAction.Params`
   as `any` and does not define built-in action params; the implementation must expose this shape.
3. Promise snapshot (JS lines 340-345, 437-442): `{error, input, output, status}` → `&xs.PromiseSnapshot[int]{...}`
   (zero `Output` 0 stands for JS `undefined` while active, since `O` = `int`).
4. Test 3 (JS lines 450-670) asserts the full unsimplified inspection event. Mapped fields: `_transitions` →
   `Transitions` (`actions` length, `eventType`, `guard` nil/non-nil, `reenter`, `source` id
   `"(machine).counting"` (JS shows `#`-prefixed toJSON form), `target` nil or ids); snapshot `children`,
   `historyValue`, `tags` asserted empty, `error`/`output` nil. `toJSON: [Function]` and `xstate$$type: 1` have
   no Go runtime equivalent and are not asserted.
5. Test 2 (JS line 222-227) passes `inspect: { next }`; ported with the `WithInspectObserver` gap. The parent
   `onDone` no-op action `() => { events; }` is ported as a no-op `ActionFunc`. JS `await waitFor(...)` resumes on
   a microtask after the synchronous emit chain; Go's `WaitFor(...).Wait()` wakes on another goroutine, so the
   test additionally waits (`require.Eventually`) for the promise actor's done snapshot, the last expected event,
   before asserting the full stream.
6. `simplifyEvents` maps unknown inspection types to `undefined`; the Go port appends `nil` for them, so any
   extra event type (e.g. `@xstate.transition`) in unfiltered tests fails the assertion as it would in JS.
7. Simplifier reads machine snapshots as `*xs.MachineSnapshot[any]` (all simplified tests use context-less
   machines, `C = any`).
