# Porting guide: @xstate/store JS tests → Go

Source of truth: `references/xstate/packages/xstate-store` (`src/*.ts`, `test/*.ts`, `src/select.test.ts`).
Go package: `github.com/nguyenvanduocit/go-xstate/store` (package `store`, dir `store/`). Tests are
`package store_test` and import it as `xstore` (a JS test variable named `store` would shadow the package)
and the engine package `github.com/nguyenvanduocit/go-xstate/xstate` as `xs`.

The Go implementation is in `store/` (`store.go`, `atom.go`, `schema.go`, `validate.go`,
`persist.go`, `undo.go`, `reset.go`, `fromstore.go`, `shallowequal.go`, `promise.go`).
`store/helpers_test.go` holds the shared test helpers.

The [core guide](core.md) applies: keep one Go test per JS test, preserve the
`// JS:` comment, and record skips and API differences in the manifest.

- Tests live beside the implementation, for example `store/atom_test.go`.
  Former numbered chunks are consolidated and no chunk build tags are required.
- Run `go vet ./store` and `go test -race -count=1 ./store` from the repository root.
- Test functions use `TestStore<Area>_<CamelCaseOfItName>`; prefix package-level
  test helpers with the area.
- Update `docs/porting/manifest/store_<area>.md` with each test's status.
- Record any missing API and its required behavior before implementing it.

## Naming rule

Go name = JS name in PascalCase (`createStore` → `CreateStore`, `StoreSnapshot` → `StoreSnapshot`).
Where Go cannot overload, the variant gets a suffix: `CreateStoreFromLogic`, `CreateComputedAtom`,
`CreateAtomConfigFunc`.

## Contract conventions (how JS maps to Go)

| JS | Go |
|---|---|
| `createStore({context, on})` | `xstore.CreateStore(xstore.StoreConfig[ctx]{Context: ctx{Count: 0}, On: map[string]xstore.StoreAssigner[ctx]{...}})`; local `type ctx struct{ Count int }` |
| `createStore({context, schemas, on})` | add `Schemas: &xstore.StoreSchemas{Context: s, Events: map[string]xstore.Schema{...}, Emitted: ...}` |
| `createStore({getInitialSnapshot, transition})` | `xstore.CreateStoreFromLogic(xstore.StoreLogic[ctx]{GetInitialSnapshot: ..., Transition: ...})` |
| `createStoreConfig(cfg)` | `xstore.CreateStoreConfig(cfg)` (identity) |
| `createStoreLogic({context: (input) => ..., selectors, on})` → `logic.createStore(input)` | `xstore.CreateStoreLogic(xstore.StoreConfig[ctx]{ContextFn: func(in any) ctx {...}, Selectors: map[string]func(ctx) any{...}, On: ...}).CreateStore(input)` |
| `store.selectors.count.get()` | `s.Selectors()["count"].Get()` (type `any`) |
| handler `(ctx, ev, enq) => ({...})` | `func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) { return ctx{...}, true }` — return the WHOLE next context |
| handler that returns `undefined` (no assignment) | `return c, false` |
| `ev.by` | `ev.(xs.E)["by"]` (payloads from `Trigger`/`Send` keep Go types: `int` stays `int`) |
| `enq.emit.increased({by: 1})` / `enq.emit.reset()` | `enq.Emit("increased", xs.E{"by": 1})` / `enq.Emit("reset")` |
| `enq.trigger.inc({by: 1})` | `enq.Trigger("inc", xs.E{"by": 1})` |
| `enq.effect(() => ...)` / `enq.effect(({trigger, send, getSnapshot}) => ...)` | `enq.Effect(func(e *xstore.StoreEffectEnqueue[ctx]) {...})`; `e.Trigger(...)`, `e.Send(ev)`, `e.GetSnapshot()` |
| async effect (`async () => { await ...; }`) | effect starts a goroutine; guard shared state (tests run with `-race`) |
| `store.send({type: 'inc', by: 1})` | `s.Send(xs.E{"type": "inc", "by": 1})` |
| `store.trigger.inc({by: 1})` / `store.trigger.inc()` | `s.Trigger("inc", xs.E{"by": 1})` / `s.Trigger("inc")` |
| `store.can.inc({by: 1})` | `s.Can("inc", xs.E{"by": 1})` |
| `store.getSnapshot()` / `store.get()` / `store.getInitialSnapshot()` | `s.GetSnapshot()` / `s.Get()` / `s.GetInitialSnapshot()` — `*xstore.StoreSnapshot[ctx]` |
| `snapshot.context`, `.status` | `snap.Context`, `snap.Status` (`xs.StatusActive`, ...) |
| extension-added snapshot keys (`snapshot[revision]`, undo history) | `snap.ExtensionState["revision"]` (copy the map before writing) |
| `store.subscribe(fn)` / `store.subscribe({next, ...})` | `s.SubscribeNext(fn)` / `s.Subscribe(xs.Observer[*xstore.StoreSnapshot[ctx]]{...})` |
| `store.on('ev', h)` / `store.on('*', h)` | `s.On("ev", func(e xs.Event) {...})` / `s.On("*", ...)`; emitted events are `xs.E` |
| `store.inspect(fn)` | `s.Inspect(func(e xstore.StoreInspectionEvent) {...})`; `e.Type == xs.InspectTransition`, `e.Event`, `e.Snapshot.(*xstore.StoreSnapshot[ctx])` |
| `store.sessionId`, `store.schemas` | `s.SessionID()`, `s.Schemas()` (`assert.Same` for `toBe(schemas)`) |
| `store.transition(snapshot, event)` → `[next, effects]` | `next, effects := s.Transition(snap, ev)` |
| `effects[0]` emitted event / `typeof effects[0] === 'function'` | `effects[0].Emitted` (e.g. `xs.E{"type": "increased", "by": 2}`) / `effects[0].Run != nil`; call `effects[i].Run(nil)` for `effect()` |
| `createStoreTransition(transitions)` | `xstore.CreateStoreTransition(map[string]xstore.StoreAssigner[ctx]{...})` returns `xstore.StoreTransition[ctx]` (result: `.Snapshot`, `.Effects`, `.Allowed`) |
| custom extension `.with((logic) => ({...logic, transition}))` | `s.With(func(l xstore.StoreLogic[ctx]) xstore.StoreLogic[ctx] { next := l; next.Transition = ...; return next })` |
| `store.select(sel, eq?)` | `xstore.Select(s, func(c ctx) T {...}, eq)` → `xstore.ReadonlyAtom[T]` |
| `shallowEqual(a, b)` | `xstore.ShallowEqual(a, b)` |
| `.with(reset())` / `reset({to})` | `.With(xstore.Reset[ctx]())` / `xstore.Reset(xstore.ResetOptions[ctx]{To: func(initial, current ctx) ctx {...}})` |
| `.with(undoRedo())` / `undoRedo({strategy: 'snapshot', historyLimit, compare, restore, getTransactionId, skipEvent})` | `.With(xstore.UndoRedo[ctx]())` / `xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{Strategy: xstore.UndoRedoSnapshot, HistoryLimit: 2, ...})`; `getTransactionId` returning `null`/`undefined` → `""` |
| `.with(validateSchemas(opts))` | `.With(xstore.ValidateSchemas[ctx](xstore.ValidateSchemasOptions{...}))`; `{context: false}` → `SkipContext: true`; `unknownEvents: 'ignore'` → `UnknownEvents: xstore.UnknownIgnore` |
| `console.warn` spy for validateSchemas | `ValidateSchemasOptions{Warn: func(args ...any) { warn.Call(args...) }}` |
| `toThrow(StoreValidationError)` | `var ve *xstore.StoreValidationError; assert.ErrorAs(t, recovered(fn).(error), &ve)` |
| `getThrown(fn)` + `toMatchObject({reason, eventType})` | `ve := recovered(fn).(*xstore.StoreValidationError)`; `assert.Equal(t, xstore.ReasonUnknownEvent, ve.Reason)` |
| `fromStore({context: (input) => ..., on})` + `createActor(logic, {input})` | `xs.CreateActor(xstore.FromStore(xstore.StoreConfig[ctx]{ContextFn: ..., On: ...}), xs.WithInput(42))`; `actor.On("increased", ...)` for emitted events |
| `createAtom(value)` / `atom.set(v)` / `atom.set(prev => ...)` | `xstore.CreateAtom(v)` / `a.Set(v)` / `a.Update(func(prev T) T {...})` |
| `createAtom((prev) => ...)` (computed) | `xstore.CreateComputedAtom(func(prev *T) T {...})` (`prev == nil` on the first run) |
| `{compare: (a, b) => ...}` | `xstore.AtomOptions[T]{Compare: func(prev, next T) bool {...}}` |
| `createAtomConfig(42).createAtom()` / `createAtomConfig((input) => ...).createAtom(input)` | `xstore.CreateAtomConfig(42).CreateAtom()` / `xstore.CreateAtomConfigFunc(func(in I) T {...}).CreateAtom(in)` |
| `createReducerAtom(init, reducer)` / `.send(ev)` | `xstore.CreateReducerAtom(init, func(s S, e E) S {...})` / `.Send(e)` |
| `createAsyncAtom(async ({signal}) => ...)` | `xstore.CreateAsyncAtom(func(ctx context.Context) (T, error) {...})`; `signal` is `ctx` |
| `{status: 'pending'}` / `{status: 'done', data}` / `{status: 'error', error}` | `xstore.AsyncAtomState[T]{Status: xstore.AsyncPending}` / `{Status: xstore.AsyncDone, Data: d}` / `{Status: xstore.AsyncError, Error: err}` |
| `atom.subscribe(fn)` / `atom.get()` | `a.SubscribeNext(fn)` / `a.Get()` (every atom and store implements `xstore.ReadonlyAtom[T]`) |
| `persist({name, storage, ...})` | `.With(xstore.Persist(xstore.PersistOptions[ctx]{Name: "test", Storage: storage, ...}))`; C needs json tags (`Count int \`json:"count"\``) so stored JSON matches JS |
| `strategy: 'event'`, `maxEvents` | `Strategy: xstore.PersistEvent, MaxEvents: 2` |
| `throttle: 100` + `vi.useFakeTimers()` / `vi.advanceTimersByTime(100)` | `Throttle: ms(100), Clock: clock` with `clock := xs.NewSimulatedClock()` / `clock.Increment(ms(100))` |
| `pick`, `merge`, `migrate`, `filter`, `serialize`, `deserialize`, `onDone`, `onError`, `skipHydration`, `version` | `Pick`, `Merge`, `Migrate`, `Filter`, `Serialize`, `Deserialize`, `OnDone`, `OnError`, `SkipHydration`, `Version`; event strategy: `MigrateEvents`, `SerializeEvents`, `DeserializeEvents` |
| `StateStorage` object literal | a type implementing `xstore.Storage`, or `xstore.StorageFuncs{GetItemFunc: ..., SetItemFunc: ..., RemoveItemFunc: ...}` |
| `createMockStorage()` / `createMemoryStorage()` / `createStorage()` | `newMemoryStorage()` (helpers) |
| `createAsyncMockStorage()` | `newAsyncMemoryStorage()` (helpers) |
| `getItem` returning `null` / a string / a Promise | `return nil, nil` / `return strPtr(v), nil` / `return nil, p` |
| `setItem`/`removeItem` sync / `async` | `return nil` / return a `*xstore.Promise[struct{}]` from `xstore.NewPromise` |
| `JSON.parse(storage.getItem('k'))` / `await storage.getItem('k')` | `storedJSON(t, storage, "k")` (numbers are `float64`) / `getItem(t, storage, "k")` |
| `JSON.stringify(v)` | `toJSON(t, v)` |
| `createJSONStorage(() => storage)` | `xstore.CreateJSONStorage(func() xstore.Storage { return storage })`; a throwing getter → `panic(...)` |
| `flushStorage(s)`, `clearStorage(s)`, `rehydrateStore(s)`, `isHydrated(s)` | `xstore.FlushStorage(s)`, `xstore.ClearStorage(s)`, `xstore.RehydrateStore(s)`, `xstore.IsHydrated(s)` |
| `clearStorage({getSnapshot: store.getSnapshot})` | `xstore.ClearStorage(s)` |
| `await flushStorage(s)` / `expect(flushStorage(s)).toBeUndefined()` | `xstore.FlushStorage(s).Wait()` (nil-safe) / `assert.Nil(t, xstore.FlushStorage(s))` |
| `createBroadcastStorage(base)` with a mocked global `BroadcastChannel` | `xstore.CreateBroadcastStorage(base, xstore.BroadcastStorageOptions{NewChannel: mockFactory})`; the mock implements `xstore.BroadcastChannel` (file-local registry) |
| broadcast message `{type: 'xstate-store-update', name}` | `xstore.BroadcastMessage{Type: "xstate-store-update", Name: "counter"}` |
| `subscribeToBroadcastStorage(s)` → unsubscribe | `unsub := xstore.SubscribeToBroadcastStorage(s)` |
| `z.object`, `z.number`, `z.string`, `z.literal`, `.optional()`, `z.number().max(n)`, `.refine(async ...)` | `zObject(map[string]*zSchema{...})`, `zNumber()`, `zString()`, `zLiteral(v)`, `zOptional(s)`, `zNumberMax(n)`, `zAsync(s)` (helpers) |
| custom Standard Schema object | `xstore.SchemaFunc(func(v any) (xstore.SchemaResult, *xstore.Promise[xstore.SchemaResult]) {...})` |

## Promises and sync/async results

- JS `T | Promise<T>` (storage `getItem`, schema `validate`) → Go `(T, *xstore.Promise[T])`; a non-nil promise
  means async and the `T` is ignored.
- JS `void | Promise<void>` (`setItem`, `removeItem`, `flushStorage`, `clearStorage`) → `*xstore.Promise[struct{}]`;
  nil means it finished synchronously. `Wait()`/`Done()` are nil-safe.
- `Promise.withResolvers()` in a storage/schema → `p, resolve, reject := xstore.NewPromise[T]()`.
- `await Promise.resolve()` / `setTimeout(r, 0)` used to let async work settle → `sleep(1)` or wait on the
  promise/signal. Prefer waiting on a concrete promise/signal when the test has one.

## Identity and ordering differences

- `toBe(snapshot)` → `assert.Same`. Preserve assertions about unchanged context keeping the same
  snapshot, and record how Go value equality or reference identity represents the JS case in the manifest.
- Object.is default equality (atoms, `select`): `==` for comparable values; maps/slices/pointers compare
  by identity. Non-nil functions compare unequal because Go does not expose closure identity;
  use a custom comparator when function values should suppress updates. Use `xstore.ShallowEqual`
  or a custom compare where JS passes one.
- Event types listed from `on`/`schemas.events` are not ordered by insertion in Go; no test observes the order.

## Untranslatable tests

- `types.test.tsx` and `// @ts-expect-error`-only checks → `t.Skip("N/A: type-level only — ...")`.
- `vue.test.ts`, `@statelyai/inspect` (`createBrowserInspector`), `Symbol.observable`, Immer `produce` →
  `t.Skip("N/A: ...")`. For Immer tests, port the runtime behaviour if it is expressible without a producer.
- Async assigners (`async (ctx) => ...` → "Async transition unsupported here") have no Go form (handlers
  are synchronous) → `N/A-runtime`.
- `localStorage` global stubbing → port with `Storage: nil` (the Go default is a no-op storage).
