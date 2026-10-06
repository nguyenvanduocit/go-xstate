> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: input_1

Source: `references/xstate/packages/core/test/input.test.ts` lines 1-330.
Go file: `xstate/input_test.go`.

Totals: 15 JS tests; 15 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| input > should create a machine with input | TestInput_ShouldCreateMachineWithInput | ported | Input is a local struct `input{StartCount}`. |
| input > initial event should have input property | TestInput_InitialEventShouldHaveInputProperty | ported | `event.input.greeting` -> `a.Event.(xs.InitEvent).Input.(input).Greeting`. |
| input > should error if input is expected but not provided | TestInput_ShouldErrorIfInputIsExpectedButNotProvided | ported | JS TypeError on `undefined.greeting` -> type assertion panic on nil input. `@ts-expect-error` dropped (type-level). |
| input > should retain the machine snapshot interface when resolving input throws | TestInput_ShouldRetainMachineSnapshotInterfaceWhenResolvingInputThrows | ported | Same throw mechanism as above. |
| input > should be a type error if input is not expected yet provided | TestInput_ShouldBeTypeErrorIfInputIsNotExpectedYetProvided | ported | `not.toThrowError()` -> `assert.NotPanics`. |
| input > should provide input data to invoked machines | TestInput_ShouldProvideInputDataToInvokedMachines | ported | Inline `src: machine` -> `InvokeConfig.Logic`. |
| input > should provide input data to spawned machines | TestInput_ShouldProvideInputDataToSpawnedMachines | ported | `spawn` inside `assign` -> `a.Spawn(..., xs.SpawnOptions{Input: ...})`. |
| input > should create a promise with input | TestInput_ShouldCreatePromiseWithInput | ported | `setTimeout(res, 5)` -> `sleep(5)`. |
| input > should create a transition function actor with input | TestInput_ShouldCreateTransitionFunctionActorWithInput | ported |  |
| input > should create an observable actor with input | TestInput_ShouldCreateObservableActorWithInput | ported | `state.context?.count !== 42` -> zero-value `Context.Count != 42`. |
| input > should create a callback actor with input | TestInput_ShouldCreateCallbackActorWithInput | ported |  |
| input > should provide a static inline input to the referenced actor | TestInput_ShouldProvideStaticInlineInputToReferencedActor | ported | Child context `{}` -> `map[string]any{}`. |
| input > should provide a dynamic inline input to the referenced actor | TestInput_ShouldProvideDynamicInlineInputToReferencedActor | ported | Dynamic input -> `xs.NewExpr`. |
| input > should call the input factory with self when invoking | TestInput_ShouldCallInputFactoryWithSelfWhenInvoking | ported | `toHaveBeenCalledWith(actor)` -> `assert.Contains(calls, []any{actor})`. |
| input > should call the input factory with self when spawning | TestInput_ShouldCallInputFactoryWithSelfWhenSpawning | ported | Same as above via `xs.SpawnChild`. |

## API gaps

None (no `apigap_input_1.go`).

## Ambiguities for reviewers

- JS lines 47-62 / 64-84: the error is produced by JS reading a property of `undefined`. In Go the
  ContextFn performs `a.Input.(input)` on a nil input, which panics; the implementation must turn a
  ContextFn panic into `Status == error` (not propagate it from `GetSnapshot`).
- JS lines 293-306 / 308-328: `toHaveBeenCalledWith(actor)` in Vitest is deep equality; the Go test
  compares `a.Self` against the root `*xs.Actor` via `assert.Contains` (reflect.DeepEqual of the same
  pointer). The implementation must pass the root `*Actor` itself (not a wrapper) as `ExprArgs.Self`.
- JS lines 182-202: the subscription callback calls `sub.Unsubscribe()`; `sub` is assigned before
  `Start()`, so it is set before the first emission.
