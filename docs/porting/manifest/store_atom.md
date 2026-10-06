> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: store_atom_1

Source: `references/xstate/packages/xstate-store/test/atom.test.ts` lines 1-1048.
Go file: `store/atom_test.go` (build tag `port_store_atom_1`, package `store_test`).

Totals: 49 JS tests; 47 ported, 2 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| creates an atom | TestStoreAtom_CreatesAnAtom | ported |  |
| creates an atom from atom config | TestStoreAtom_CreatesAnAtomFromAtomConfig | ported |  |
| creates an atom from atom config and input | TestStoreAtom_CreatesAnAtomFromAtomConfigAndInput | ported | `CreateAtomConfigFunc[T, I]` with a local `input` struct. |
| sets the value of the atom using a function | TestStoreAtom_SetsTheValueOfTheAtomUsingAFunction | ported | `atom.set(fn)` -> `Update`. |
| does not subscribe a writable atom to reads inside its updater | TestStoreAtom_DoesNotSubscribeAWritableAtomToReadsInsideItsUpdater | ported | `observer.mock.calls` -> `spy.Calls()`. |
| drains notifications before rethrowing the first subscriber error | TestStoreAtom_DrainsNotificationsBeforeRethrowingTheFirstSubscriberError | ported | `toThrow(error)` -> `assert.PanicsWithValue(err, ...)` (identity of the panic value; subscriber panics with the same `error`). |
| rethrows undefined and still delivers reentrant notifications | TestStoreAtom_RethrowsUndefinedAndStillDeliversReentrantNotifications | ported | Go cannot panic with `undefined`; subscriber panics with sentinel `storeAtom1Undefined{}` and the test asserts the first thrown value is rethrown unchanged (`assert.Equal(sentinel, thrown)`). `didThrow` kept. |
| can set the value to undefined | TestStoreAtom_CanSetTheValueToUndefined | ported | `CreateAtom[any](1)`; `nil` stands for `undefined`. |
| can subscribe to atom changes | TestStoreAtom_CanSubscribeToAtomChanges | ported | `toHaveBeenCalledWith` -> `assert.Contains(log.Calls(), []any{v})`. |
| can unsubscribe from atom changes | TestStoreAtom_CanUnsubscribeFromAtomChanges | ported |  |
| can create a combined atom | TestStoreAtom_CanCreateACombinedAtom | ported | `CreateComputedAtom`; trailing `numAtom.set(5)` kept (no assertion in JS). |
| allows updates to a computed dependency during a subscription callback | TestStoreAtom_AllowsUpdatesToAComputedDependencyDuringASubscriptionCallback | ported | Atom value is `*abc` (pointer) so every `{...ctx}` is a distinct object under the default Object.is equality, matching JS (a comparable struct value would suppress equal sets). |
| does not loop when updating a computed dependency which affects an atoms own state | TestStoreAtom_DoesNotLoopWhenUpdatingAComputedDependencyWhichAffectsAnAtomsOwnState | ported |  |
| works with a mix of atoms and stores | TestStoreAtom_WorksWithAMixOfAtomsAndStores | ported | `store.send({type:'nameUpdated', name})` -> `Send(xs.E{...})`. |
| works with stores | TestStoreAtom_WorksWithStores | ported | Two stores with distinct context types. |
| works with selectors | TestStoreAtom_WorksWithSelectors | ported | `store.select(fn)` -> `xstore.Select(store, fn)`. |
| allows sending events to the store during a selector subscription | TestStoreAtom_AllowsSendingEventsToTheStoreDuringASelectorSubscription | ported |  |
| works with selectors (get API) | TestStoreAtom_WorksWithSelectorsGetAPI | ported | Identical body to `works with selectors` in JS. |
| combined atoms should be read-only | TestStoreAtom_CombinedAtomsShouldBeReadOnly | ported | `set?.(2)` -> optional call via reflection (`MethodByName("Set")`), skipped when absent (the Go `ReadonlyAtom` has no `Set`); value assertions kept. |
| combined atom getters accept only prev as an argument | TestStoreAtom_CombinedAtomGettersAcceptOnlyPrevAsAnArgument | N/A-type | Only a `@ts-expect-error` signature check; the second statement merely creates an atom with no expectation. |
| conditionally read atoms are properly read in combined atoms | TestStoreAtom_ConditionallyReadAtomsAreProperlyReadInCombinedAtoms | ported |  |
| conditionally read atoms are properly unsubscribed when no longer needed | TestStoreAtom_ConditionallyReadAtomsAreProperlyUnsubscribedWhenNoLongerNeeded | ported | Computed atom of type `any`; `{}` -> a fresh `map[string]any{}` per computation (identity-compared, so each recompute notifies like JS). |
| handles diamond dependencies with single update | TestStoreAtom_HandlesDiamondDependenciesWithSingleUpdate | ported |  |
| handles complex diamond dependencies correctly | TestStoreAtom_HandlesComplexDiamondDependenciesCorrectly | ported |  |
| supports custom equality functions through compare option | TestStoreAtom_SupportsCustomEqualityFunctionsThroughCompareOption | ported | `toHaveBeenLastCalledWith` -> last element of `Calls()`. |
| uses Object.is as default equality function | TestStoreAtom_UsesObjectIsAsDefaultEqualityFunction | ported | Atom of `*obj` so identity semantics apply. |
| Atom-specific properties should not be exposed | TestStoreAtom_AtomSpecificPropertiesShouldNotBeExposed | N/A-type | Only bare property accesses under `@ts-expect-error`; no runtime expectation. |
| computed atoms can use their previous value in the getter | TestStoreAtom_ComputedAtomsCanUseTheirPreviousValueInTheGetter | ported | `prev ?? 0` -> `prev == nil`. |
| reducer atoms > updates from current state and sent event | TestStoreAtom_ReducerAtoms_UpdatesFromCurrentStateAndSentEvent | ported |  |
| reducer atoms > can receive arbitrary event values | TestStoreAtom_ReducerAtoms_CanReceiveArbitraryEventValues | ported | Event type `any`; JS `state + event` -> `state + fmt.Sprint(event)`. |
| reducer atoms > notifies subscribers when the reducer changes state | TestStoreAtom_ReducerAtoms_NotifiesSubscribersWhenTheReducerChangesState | ported | `toHaveBeenNthCalledWith` -> indexing `Calls()`. |
| reducer atoms > can be used by derived atoms | TestStoreAtom_ReducerAtoms_CanBeUsedByDerivedAtoms | ported |  |
| reducer atoms > does not track atom reads inside the reducer | TestStoreAtom_ReducerAtoms_DoesNotTrackAtomReadsInsideTheReducer | ported |  |
| async atoms > should recompute after a dependency changes following %s settlement | TestStoreAtom_AsyncAtoms_ShouldRecomputeAfterADependencyChangesFollowingSettlement | ported | `it.each(['done','error'])` -> one Go test with `t.Run` per status. `await Promise.resolve()` -> poll until settled, then `sleep(20)` so the notification drains before `observer.Reset()` / the final assertion. The getter reads `count.Get()` on the atom's goroutine (dependency tracking from that goroutine is the implementation's concern). |
| async atoms > should recompute lazily after a settled dependency changes | TestStoreAtom_AsyncAtoms_ShouldRecomputeLazilyAfterASettledDependencyChanges | ported | `vi.fn` getter -> `spy.Call()` inside the getter. |
| async atoms > should retain dependencies when the async comparator suppresses a result | TestStoreAtom_AsyncAtoms_ShouldRetainDependenciesWhenTheAsyncComparatorSuppressesAResult | ported | `toHaveBeenCalledExactlyOnceWith` -> `Calls() == [[state]]`. `await Promise.resolve()` for the suppressed cases -> `sleep(50)`. |
| async atoms > async atoms should work (fulfilled) | TestStoreAtom_AsyncAtoms_AsyncAtomsShouldWorkFulfilled | ported | `setTimeout(0)` -> poll until not pending. |
| async atoms > async atoms should work (rejected) | TestStoreAtom_AsyncAtoms_AsyncAtomsShouldWorkRejected | ported | `expect.any(Error)` -> `assert.Equal(AsyncError, Status)` + `assert.Error(state.Error)`. |
| async atoms > should only call getValue once for multiple concurrent reads | TestStoreAtom_AsyncAtoms_ShouldOnlyCallGetValueOnceForMultipleConcurrentReads | ported | Call counter is `atomic.Int32` (getter runs on its own goroutine). |
| async atoms > should only call getValue once even when error occurs | TestStoreAtom_AsyncAtoms_ShouldOnlyCallGetValueOnceEvenWhenErrorOccurs | ported | Same as above; `expect.any(Error)` -> `assert.Error`. |
| async atoms > async atoms should not have a .set() method | TestStoreAtom_AsyncAtoms_AsyncAtomsShouldNotHaveASetMethod | ported | `'set' in atom` -> `reflect.TypeOf(atom).MethodByName("Set")` is absent. |
| async atoms > should pass an abort signal to async atoms | TestStoreAtom_AsyncAtoms_ShouldPassAnAbortSignalToAsyncAtoms | ported | The in-getter `expect(signal).toBeInstanceOf(AbortSignal)` would be swallowed into an error state in JS; Go reports the received `context.Context` over a channel and asserts it is non-nil and not cancelled (fails with a timeout if getValue is never called). |
| async atoms > should abort and ignore stale async atom results | TestStoreAtom_AsyncAtoms_ShouldAbortAndIgnoreStaleAsyncAtomResults | ported | `signals[i].aborted` -> `ctx.Err() != nil`. Because the getter runs on a goroutine, the test waits until each run has registered before the next step (JS registers synchronously). Resolvers are channels closed to resolve. |
| async atoms > should ignore stale async atom errors | TestStoreAtom_AsyncAtoms_ShouldIgnoreStaleAsyncAtomErrors | ported | Resolve/reject modelled by a buffered `chan error` per run (nil = resolve, non-nil = reject); the `signal.aborted` check inside `.then` is kept via `ctx.Err()`. |
| async atoms > should notify subscribers when async operation completes successfully | TestStoreAtom_AsyncAtoms_ShouldNotifySubscribersWhenAsyncOperationCompletesSuccessfully | ported | Getter delay 10ms as in JS; the wait is 100ms instead of 20ms to absorb goroutine scheduling jitter. |
| async atoms > should notify subscribers when async operation fails | TestStoreAtom_AsyncAtoms_ShouldNotifySubscribersWhenAsyncOperationFails | ported | Same timing note; error identity compared via `AsyncAtomState{Status: AsyncError, Error: failure}`. |
| async atoms > should notify multiple subscribers when async operation completes | TestStoreAtom_AsyncAtoms_ShouldNotifyMultipleSubscribersWhenAsyncOperationCompletes | ported | Same timing note. |
| async atoms > subscribe callback should not track dependencies from .get() calls | TestStoreAtom_AsyncAtoms_SubscribeCallbackShouldNotTrackDependenciesFromGetCalls | ported | Atom value `[]int` (slices compare by identity, so each `Set` is a change, as with JS arrays); `ids` is a computed string. `.sort().join(',')` -> `storeAtom1SortedJoin`. |
| async atoms > subscribe callback should not track deps on non-computed atoms | TestStoreAtom_AsyncAtoms_SubscribeCallbackShouldNotTrackDepsOnNonComputedAtoms | ported | Same notes. |

## API gaps

None (`store/apigap_store_atom_1.go` not created).

## Ambiguities for the reviewer

- Async atom getters run on their own goroutine (contract in `store/atom.go`). The JS tests read dependencies (`count.get()`) synchronously inside the async function before its first `await`; the Go port reads them in the getter goroutine, so the implementation must track dependencies reads made there. Tests that depend on it: the three `it.each`/lazy/comparator recompute tests and both stale-result tests.
- `await Promise.resolve()` (one microtask) has no Go equivalent; settle points use polling (`storeAtom1WaitUntil`, 2s cap) and short `sleep`s to let notifications drain. No assertion was dropped.
- `throw undefined` is represented by a sentinel value (see the "rethrows undefined" row); the implementation must rethrow the first thrown value unchanged, whatever it is.
- Default `Object.is` equality for atoms holding structs: tests that need "every set is a change" use pointer values (identity), per the contract in `store/atom.go`.
