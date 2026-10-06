> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: emit_1

Source: `references/xstate/packages/core/test/emit.test.ts` lines 1-484 (whole file).
Go file: `xstate/emit_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| event emitter > only emits expected events if specified in setup | TestEmit_OnlyEmitsExpectedEventsIfSpecifiedInSetup | N/A-type | JS L27. Only `@ts-expect-error` checks (emit restricted to `types.emitted`); no runtime expectations |
| event emitter > emits any events if not specified in setup (unsafe) | TestEmit_EmitsAnyEventsIfNotSpecifiedInSetupUnsafe | ported | JS L46. No `expect`; the implicit check (createMachine does not throw) → `assert.NotPanics` |
| event emitter > emits events that can be listened to on actorRef.on(…) | TestEmit_EmitsEventsThatCanBeListenedToOnActorRefOn | ported | JS L57. `setTimeout(send)` → `time.AfterFunc(0, send)` registered after `On`; awaited promise → buffered channel + 2s timeout |
| event emitter > enqueue.emit(…) emits events that can be listened to on actorRef.on(…) | TestEmit_EnqueueEmitEmitsEventsThatCanBeListenedToOnActorRefOn | ported | JS L83. `enqueue.emit(ev)` → `a.Enqueue(xs.Emit(ev))`; `@ts-expect-error` on `type: 'unknown'` dropped (runtime emit kept) |
| event emitter > handles errors | TestEmit_HandlesErrors | ported | JS L115. Listener `throw` → `panic(errors.New("oops"))`; mocked `reportUnhandledError` → `xs.WithUnhandledErrorHandler` recording to a spy (not asserted, JS asserts nothing on it) |
| event emitter > dynamically emits events that can be listened to on actorRef.on(…) | TestEmit_DynamicallyEmitsEventsThatCanBeListenedToOnActorRefOn | ported | JS L143. `emit(({context}) => ...)` → `xs.Emit(xs.NewExpr(...))`; `toEqual` → `assert.Equal(xs.E{...}, event)` |
| event emitter > listener should be able to read the updated snapshot of the emitting actor | TestEmit_ListenerShouldBeAbleToReadUpdatedSnapshotOfEmittingActor | ported | JS L170 |
| event emitter > wildcard listeners should be able to receive all emitted events | TestEmit_WildcardListenersShouldBeAbleToReceiveAllEmittedEvents | ported | JS L200. `satisfies` / `@ts-expect-error` type checks dropped |
| event emitter > events can be emitted from promise logic | TestEmit_EventsCanBeEmittedFromPromiseLogic | ported | JS L231. JS emits synchronously inside `start()`; `xs.FromPromise` runs fn on its own goroutine, so the Go test waits on a signal resolved by the listener before the `objectContaining` assertion. Type checks dropped |
| event emitter > events can be emitted from transition logic | TestEmit_EventsCanBeEmittedFromTransitionLogic | ported | JS L265. Initial state `{}` → `map[string]any{}`; type checks dropped |
| event emitter > events can be emitted from observable logic | TestEmit_EventsCanBeEmittedFromObservableLogic | ported | JS L307. Inert `{subscribe: () => ({unsubscribe})}` → `&observable[any]{...}` from helpers_test.go returning a no-op teardown |
| event emitter > events can be emitted from event observable logic | TestEmit_EventsCanBeEmittedFromEventObservableLogic | ported | JS L349 |
| event emitter > events can be emitted from callback logic | TestEmit_EventsCanBeEmittedFromCallbackLogic | ported | JS L395 |
| event emitter > events can be emitted from callback logic (restored root) | TestEmit_EventsCanBeEmittedFromCallbackLogicRestoredRoot | ported | JS L428. `children.cb!.on(...)` → `require.NotNil` + `xs.As[*xs.CallbackSnapshot](cb).On(...)` (ActorRef has no `On`) |

Totals: 14 JS tests — 13 ported, 1 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. `apigap_emit_1.go` not created.

## Ambiguities for review

- JS L231-262 (promise logic): the JS assertion runs synchronously right after `actor.start()`. The Go contract (`logic.go` `FromPromise`: "fn runs on its own goroutine") makes emission asynchronous, so the test waits (2s timeout) for the listener before asserting. The assertion itself is unchanged.
- JS L27-44: classified N/A-type because the only checks are `@ts-expect-error`. Its runtime side (createMachine with emit actions does not throw) is covered by the next test, L46.
- `expect.objectContaining` is implemented by file-local helper `emit1AssertCalledWithContaining` (any call whose single argument is an `xs.E` containing every expected key/value), used by 6 tests.
- Child `On` on a restored root (L428): uses `xs.As[*xs.CallbackSnapshot]`; this assumes `As` can recover the typed callback actor from `Children["cb"]` before the root is started.
