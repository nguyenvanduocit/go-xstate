> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: event_1

Source: `references/xstate/packages/core/test/event.test.ts` lines 1-138.
Go file: `xstate/event_test.go`.

Totals: 2 JS tests; 2 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| events > should be able to respond to sender by sending self | TestEvent_ShouldBeAbleToRespondToSenderBySendingSelf | ported | `expect(event.sender).toBeDefined()` inside the sendTo target expr → `assert.NotNil(t, sender)` inside `xs.NewExpr`. `types.events` dropped (type-only). `invoke.src: authServerMachine` → `InvokeConfig.Logic`. |
| nested transitions > only take the transition of the most inner matching event | TestEvent_NestedTransitions_OnlyTakeTheTransitionOfTheMostInnerMatchingEvent | ported | Context `{email, password}` → local struct `signInContext{Email, Password}`. |

## API gaps

None (no `apigap_event_1.go`).

## Ambiguities

- JS line 11-61: `sender: self` is put into the event as an `xs.ActorRef` under key `"sender"`; the delayed `sendTo` target Expr returns it as `any`. Implementation must accept an `ActorRef` returned from an Expr target (contract `action.go` SendTo doc says it does).
- JS line 20-25: the target expr runs inside the child actor; with a 10ms delay the assertion may run on a timer goroutine — `assert.NotNil` (not `require`) is goroutine-safe.
