> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# Test translation fixes

One line per edit: `file:line · JS file:line · what was wrong`.

- actor_logic_test.go:460 · references/xstate/packages/core/test/actorLogic.test.ts:355-361,398-401 · `deferredList` is pushed inside the promise creator, i.e. in actor START order (promise.ts `start`), not creation order; `StateMachine.start` (StateMachine.ts:505) starts the surviving `children.p` (the p2 invoke) before the deferred spawn-starts, so JS `deferredList[0]` is the p2-region actor. The port recovered order from `@xstate.actor` (creation) events; it now uses the `xstate.init` `@xstate.event` emitted on start (createActor.ts:545).
- rehydration_test.go:168 · references/xstate/packages/core/test/rehydration.test.ts:178-211 · the JS body never awaits, so `Promise.resolve(11)` (a microtask) cannot settle before the assertions; the port's promise body returned immediately on its own goroutine and could settle between `Start()`, `GetPersistedSnapshot()` and `Send(NEXT)` (flaky: value `b`). The promise body now blocks until the test body returns.
