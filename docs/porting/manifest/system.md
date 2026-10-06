> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: system_1

Source: `references/xstate/packages/core/test/system.test.ts` lines 1-627 (whole file; 22 `it()` calls, no `it.each`/`it.skip`/`it.todo`, single `describe('system')`).
Go file: `xstate/system_test.go`. Gap file: `apigap_system_1.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| system > should register an invoked actor | TestSystem_ShouldRegisterAnInvokedActor | ported | `return promise` → `sig.Wait(t)`; `system?.get` → nil check on `a.System`. Type-only `MySystem` alias dropped. |
| system > should register a spawned actor | TestSystem_ShouldRegisterASpawnedActor | ported | `throw new Error('no')` → `panic(errors.New("no"))`. |
| system > system can be immediately accessed outside the actor | TestSystem_SystemCanBeImmediatelyAccessedOutsideTheActor | ported | |
| system > root actor can be given the systemId | TestSystem_RootActorCanBeGivenTheSystemId | ported | `toBe(actor)` → `assert.Same`. |
| system > should remove invoked actor from receptionist if stopped | TestSystem_ShouldRemoveInvokedActorFromReceptionistIfStopped | ported | `toBeUndefined` → `assert.Nil`. |
| system > should remove spawned actor from receptionist if stopped | TestSystem_ShouldRemoveSpawnedActorFromReceptionistIfStopped | ported | |
| system > should throw an error if an actor with the system ID already exists | TestSystem_ShouldThrowAnErrorIfAnActorWithTheSystemIDAlreadyExists | ported | Inline snapshot `[[Error: Actor with system ID 'test' already exists.]]` → exactly one call with one `error` arg with that message. |
| system > should cleanup stopped actors | TestSystem_ShouldCleanupStoppedActors | ported | `not.toThrow` → `assert.NotPanics`. |
| system > should be accessible in inline custom actions | TestSystem_ShouldBeAccessibleInInlineCustomActions | ported | |
| system > should be accessible in referenced custom actions | TestSystem_ShouldBeAccessibleInReferencedCustomActions | ported | |
| system > should be accessible in assign actions | TestSystem_ShouldBeAccessibleInAssignActions | ported | JS assigner returns undefined → Go returns `a.Context` unchanged. |
| system > should be accessible in sendTo actions | TestSystem_ShouldBeAccessibleInSendToActions | ported | |
| system > should be accessible in promise logic | TestSystem_ShouldBeAccessibleInPromiseLogic | ported | `expect.assertions(2)` → atomic counter == 2; waits on a signal because the Go promise body runs on a goroutine. |
| system > should be accessible in transition logic | TestSystem_ShouldBeAccessibleInTransitionLogic | ported | `expect.assertions(2)` → counter; `get('reducer')!` → `require.NotNil`. |
| system > should be accessible in observable logic | TestSystem_ShouldBeAccessibleInObservableLogic | ported | `expect.assertions(2)` → counter; `of(0)` → `rxOf(0)`. |
| system > should be accessible in event observable logic | TestSystem_ShouldBeAccessibleInEventObservableLogic | ported | `expect.assertions(2)` → counter. |
| system > should be accessible in callback logic | TestSystem_ShouldBeAccessibleInCallbackLogic | ported | `expect.assertions(2)` → counter. |
| system > should gracefully handle re-registration of a `systemId` during a reentering transition | TestSystem_ShouldGracefullyHandleReRegistrationOfASystemIdDuringAReenteringTransition | ported | `get('listener')!` → `require.NotNil`. |
| system > should be able to send an event to an ancestor with a registered `systemId` from an initial entry action | TestSystem_ShouldBeAbleToSendAnEventToAnAncestorWithARegisteredSystemIdFromAnInitialEntryAction | ported | |
| system > system ID should be accessible on the actor | TestSystem_SystemIDShouldBeAccessibleOnTheActor | ported | Uses gap `Actor.SystemID()`. |
| system > should give a list of runnings actors | TestSystem_ShouldGiveAListOfRunningsActors | ported | Uses gap `ActorSystem.GetAll()`; `toEqual({})` → `assert.Empty`. |
| system > should unregister nested child systemIds when stopping a parent actor | TestSystem_ShouldUnregisterNestedChildSystemIdsWhenStoppingAParentActor | ported | `not.toThrow` → `assert.NotPanics`. |

Totals: 22 JS tests; 22 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_system_1.go`)

- `func (a *Actor[S]) SystemID() string` — mirrors `actor.systemId` (JS line 549).
- `func (s *ActorSystem) GetAll() map[string]ActorRef` — mirrors `system.getAll()` (JS lines 579, 586).

## Reviewer notes

- JS lines 27-31, 72-76: `type MySystem = ActorSystem<{...}>` is a type-only cast; dropped (no runtime effect).
- JS line 362/384/411/434/457 `expect.assertions(2)`: translated as an atomic counter incremented next to each original `expect`, checked at the end. The promise test (line 361) additionally waits for the promise body because `FromPromise` runs on its own goroutine in Go (JS runs the creator synchronously at start).
- JS line 236: inline snapshot of `errorSpy.mock.calls` asserted as length 1 × 1 arg, arg is `error` with exact message.
- JS line 586: `toEqual({})` accepts nil or empty map → `assert.Empty`.
