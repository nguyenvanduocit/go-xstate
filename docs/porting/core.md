# Porting guide: XState JS tests → Go

Source of truth: `references/xstate` (statelyai/xstate @ `38dcaff`, core v5.33.2).
Go module: `github.com/nguyenvanduocit/go-xstate`, engine package `xstate/`. Tests import it as `xs`.

The engine lives in `xstate/`; import it as
`xs "github.com/nguyenvanduocit/go-xstate/xstate"`. See the
[architecture map](../ARCHITECTURE.md) for implementation files.
`xstate/action_test.go` shows translated tests, and `xstate/helpers_test.go`
holds shared test helpers.

## Goal of translation

Every JS `it(...)`/`test(...)` becomes exactly one Go test with the same assertions.
Do not weaken, merge, or drop assertions. Account for every JS test in its
subject's manifest, including explicit reasons for skipped tests.

## Test layout and checks

Tests live beside the implementation in `xstate/`, using `package xstate_test`
and subject-based names such as `action_test.go` or `invoke_test.go`. The former
numbered chunks have been consolidated; no chunk build tags are required.

From the repository root:

```sh
go vet ./xstate
go test -race -count=1 ./xstate
```

For a focused change, add `-run '^TestActions_'` or the relevant test name.
The full repository check is `./scripts/test.sh`, which includes the examples
module.

## Missing API

If a faithful test needs an API that does not exist, record the upstream API,
required behavior, and a minimal failing test in the manifest. Implement and
verify the behavior before marking the port complete. Historical `apigap_*`
stubs and chunk tags in manifests describe the original translation phase.

## Naming

- Test func: `Test<Area>_<CamelCaseOfItName>`; prepend the describe path when needed for
  uniqueness (`TestActions_Entry_ShouldRunInOrder`). Max ~100 chars.
- Directly above each test: `// JS: <describe> > <nested describe> > <it name>` (verbatim names).
- Add `// JS test: https://github.com/statelyai/xstate/blob/<commit>/<path>#L<line>`
  pointing to the individual test declaration in the pinned reference commit.
  Generated tests link the shared declaration and their input row. Label a
  Go-only regression explicitly and link related implementation evidence.
- File-local helper funcs/types/vars must be prefixed with the area to avoid collisions within
  the package: e.g. `actions2Logs`, `type invoke1Ctx struct{...}`. Prefer types and
  helpers declared inside the test func.

## Contract conventions (how JS maps to Go)

| JS | Go |
|---|---|
| `createMachine({...})` | `xs.CreateMachine(xs.MachineConfig[C]{...}, impl...)`. `C` is the context type; use `struct{}` for an empty context that must serialize as `{}`. |
| `setup({...}).createMachine({...})` | `xs.NewSetup[C](xs.Implementations{...}).CreateMachine(xs.MachineConfig[C]{...})` |
| `context: {count: 0}` | `Context: ctx{Count: 0}` with a local `type ctx struct{ Count int }`. Use `map[string]any` only when keys are dynamic. |
| `context: ({input}) => ...` | `ContextFn: func(a xs.ContextArgs) ctx {...}` |
| `states: {a: {...}, b: {...}}` | `States: xs.States{{Key: "a", ...}, {Key: "b", ...}}` (document order preserved) |
| `type: 'final'`, `'parallel'`, `'history'` | `Type: xs.Final` / `xs.Parallel` / `xs.History`; `history: 'deep'` → `History: xs.Deep` |
| `on: {EV: 'b'}` | `On: map[string]xs.Transitions{"EV": {{Target: "b"}}}` |
| `on: {EV: {target: ['.a', '.b']}}` | `{{Targets: []string{".a", ".b"}}}` |
| `on: {EV: [{guard, target}, {target}]}` | `{"EV": {{Guard: g, Target: "x"}, {Target: "y"}}}` |
| `on: {EV: undefined}` (forbidden) | `{"EV": nil}` |
| `after: {1000: 'b', myDelay: 'c'}` | `After: map[string]xs.Transitions{"1000": {{Target: "b"}}, "myDelay": {{Target: "c"}}}` |
| `always: 'b'` | `Always: xs.Transitions{{Target: "b"}}` |
| `entry: [a, b]` / `entry: a` | `Entry: xs.Actions{a, b}` / `Entry: xs.Actions{a}` |
| inline action `({context, event}) => ...` | `xs.ActionFunc(func(a xs.ActionArgs[ctx]) {...})` (params: `a.Params`) |
| `'doThing'` / `{type: 'doThing', params}` | `xs.ActionRef{Type: "doThing"}` / `xs.ActionRef{Type: "doThing", Params: p}` |
| inline guard | `xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {...})` |
| named guard | `xs.GuardRef{Type: "isOk", Params: p}` |
| `and([...])`, `or`, `not`, `stateIn('#x')` | `xs.And(...)`, `xs.Or(...)`, `xs.Not(g)`, `xs.StateIn("#x")` |
| `assign({count: ({context}) => context.count + 1})` | `xs.Assign(func(a xs.AssignArgs[ctx]) ctx { c := a.Context; c.Count++; return c })` — returns the WHOLE next context; never mutate maps/slices of the previous context in place. |
| value-or-function args (input, output, params, ids, delays, send target/event) | static value, or `xs.NewExpr(func(a xs.ExprArgs[ctx]) any {...})` |
| `raise(ev)`, `sendTo(t, ev, {id, delay})` | `xs.Raise(ev)`, `xs.SendTo(t, ev, xs.SendOptions{ID: "x", Delay: 100 * time.Millisecond})` (delay names: `Delay: "myDelay"`) |
| `sendParent`, `forwardTo`, `log`, `cancel`, `stopChild`, `spawnChild`, `emit`, `enqueueActions` | `xs.SendParent`, `xs.ForwardTo`, `xs.Log`, `xs.Cancel`, `xs.StopChild`, `xs.SpawnChild`, `xs.Emit`, `xs.EnqueueActions(func(a xs.EnqueueArgs[ctx]) { a.Enqueue(...); a.Check(g) })` |
| `{ type: 'EV' }` | `xs.Ev("EV")` |
| `{ type: 'EV', value: 3 }` | `xs.E{"type": "EV", "value": 3}`; read with `a.Event.(xs.E)["value"]` |
| built-in events (`xstate.done.actor.x`, ...) | `xs.DoneActorEvent{ActorID, Output}`, `xs.ErrorActorEvent`, `xs.DoneStateEvent`, `xs.SnapshotEvent`, `xs.InitEvent`, `xs.StopEvent` |
| `invoke: {src, id, input, onDone, onError}` | `Invoke: []xs.InvokeConfig{{Src: "name" or Logic: logic, ID: "x", Input: ..., OnDone: xs.Transitions{...}}}` |
| implementations (`actions`, `guards`, `actors`, `delays`) | `xs.Implementations{Actions: map[string]xs.Action{...}, Guards: ..., Actors: map[string]xs.ActorLogic{...}, Delays: map[string]any{"d": 100 * time.Millisecond}}` |
| `machine.provide({...})` | `machine.Provide(xs.Implementations{...})` |
| `createActor(logic, {input, id, systemId, snapshot, inspect, clock, logger})` | `xs.CreateActor(logic, xs.WithInput(..), xs.WithID(..), xs.WithSystemID(..), xs.WithSnapshot(..), xs.WithInspect(..), xs.WithClock(..), xs.WithLogger(..))` |
| `actor.start()/stop()/send()/getSnapshot()` | `actor.Start()/Stop()/Send()/GetSnapshot()` (typed snapshot) |
| `actor.subscribe(fn)` / `actor.subscribe({next, error, complete})` | `actor.SubscribeNext(fn)` / `actor.Subscribe(xs.Observer[S]{Next:.., Error:.., Complete:..})` |
| `actor.on('ev', h)` | `actor.On("ev", func(e xs.Event) {...})` |
| `snapshot.value` | `snap.Value` — a `string` or `map[string]any` (nested values are `string`/`map[string]any`) |
| `snapshot.matches('a.b')`, `hasTag`, `can` | `snap.Matches("a.b")`, `snap.HasTag("t")`, `snap.Can(ev)` |
| `snapshot.children.foo` | `snap.Children["foo"]` (type `xs.ActorRef`); typed: `xs.As[*xs.MachineSnapshot[cctx]](ref)` or helper `machineSnap[cctx](ref)` |
| `snapshot.tags` (Set) | `snap.Tags` sorted `[]string` → compare with `assert.ElementsMatch` |
| `fromPromise(async ({input, signal}) => ...)` | `xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (O, error) {...})` — `signal` is `ctx` |
| `fromCallback(({sendBack, receive, input}) => cleanup)` | `xs.FromCallback(func(a xs.CallbackArgs) func() {...})` |
| `fromObservable(() => interval(10))` | `xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })` |
| `fromTransition(reducer, initial)` | `xs.FromTransition(func(s T, e xs.Event, scope *xs.ActorScope) T {...}, func(a xs.TransitionInitArgs) T {...})` |
| custom logic object literal | `&xs.Logic[S]{Transition: ..., GetInitialSnapshot: ..., Start: ...}`; `xs.BasicSnapshot[T]` is a ready-made snapshot type |
| `waitFor(actor, pred, {timeout})` | `xs.WaitFor(context.Background(), actor, pred, xs.WaitForOptions{Timeout: ...}).Wait()` returns `(S, error)`; create it BEFORE sending if JS creates the promise before sending |
| `toPromise(actor)` | `xs.ToPromise(actor).Wait()` |
| `new SimulatedClock()`; `clock.increment(100)` | `xs.NewSimulatedClock()`; `clock.Increment(100 * time.Millisecond)` |
| `inspect` events | `xs.InspectionEvent{Type: xs.InspectSnapshot, ...}` (single struct for all variants) |
| `transition(machine, s, e)`, `initialTransition` | `xs.Transition(m, s, e)`, `xs.InitialTransition(m, input...)` |

## Errors, throws, warnings

- An action/guard/actor that `throw`s in JS → `panic(err)` in Go (use `errors.New("msg")` when JS throws `new Error('msg')`, otherwise panic with the same value). Snapshot `Error` holds the recovered value.
- A promise that rejects → return a non-nil `error` (or `errors.New(...)`).
- `expect(() => fn()).toThrow('msg')` → `assert.PanicsWithError(t, "msg", fn)` if the exact full message is known, else `assert.Panics` plus a substring check with `panicMessage(fn)` (helper below).
- `reportUnhandledError` spies → `xs.WithUnhandledErrorHandler(func(err any){...})` on the root actor.
- `console.warn` spies → `xs.WithWarnHandler(func(args ...any) {...})`; unhandled errors use `xs.WithUnhandledErrorHandler`. These are separate from `WithLogger`.

## Async

- `Promise.withResolvers()` → `sig := newSignal()`; `resolve()` → `sig.Resolve()`; `await promise` → `sig.Wait(t)`.
- `await sleep(n)` / `setTimeout(r, n)` → `sleep(n)` / `time.AfterFunc(ms(n), ...)`.
- `vi.useFakeTimers()` + `vi.advanceTimersByTime(n)` → create the actor with `xs.WithClock(clock)` using `xs.NewSimulatedClock()` and call `clock.Increment(ms(n))`.
- `vi.fn()` → `newSpy()`; `toHaveBeenCalledTimes(n)` → `assert.Equal(t, n, s.Count())`; `toHaveBeenCalledWith(x)` → inspect `s.Calls()`.
- rxjs `of`, `from`, `EMPTY`, `throwError`, `interval`, `map`, `filter`, `take`, `BehaviorSubject` → `rxOf`, `rxFrom`, `rxEmpty`, `rxThrowError`, `rxInterval`, `rxMap`, `rxFilter`, `rxTake`, `newBehaviorSubject` in `helpers_test.go`.
- Things mutated from multiple goroutines (promise/timer callbacks) must be guarded (`sync.Mutex`, `atomic`) — tests run with `-race`.

## Assertions

Use `github.com/stretchr/testify/assert` (and `require` where JS would abort). `toEqual`/`toBe`/`toStrictEqual` → `assert.Equal(t, expected, actual)`; object identity (`toBe` on objects) → `assert.Same`. `toMatchInlineSnapshot` → assert the exact value shown in the inline snapshot.

## Ordering differences

Go maps have no insertion order. Where JS output order derives from object-key insertion order of `on`/`after`/implementations (e.g. `machine.events`, `stateNode.ownEvents`), the Go API returns sorted values: write the expectation sorted. `states` keep document order. Everything else (action execution order, log order, entry/exit order) must match JS exactly.

## Untranslatable tests

- Pure TypeScript type-level tests (`expectTypeOf`, `// @ts-expect-error` as the only assertion, `types: {} as ...` checks with no runtime expectations): write the Go test func with `t.Skip("N/A: type-level only — <what it checked>")`. If the test also has runtime expectations, translate those and drop only the type assertion.
- Tests specific to JS runtime features with no Go meaning (e.g. `Symbol.observable`, React/Vue bindings, `JSON.stringify` of functions): `t.Skip("N/A: <reason>")`. Be conservative: if Go can express the behaviour, port it.
- `it.skip` / `it.todo` in JS: keep as `t.Skip("skipped in JS")`.

## Manifest (required)

Update `docs/porting/manifest/<area>.md` with one row per JS test in the subject:

```
| JS test (describe > it) | Go test | Status | Notes |
```
Status ∈ `ported`, `N/A-type`, `N/A-runtime`, `skipped-in-JS`. After the table, list unresolved API gaps and any ambiguity a reviewer should check.
