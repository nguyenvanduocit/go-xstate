> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# order_1 manifest (core/test/order.test.ts lines 1-87)

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| document order > should specify the correct document order for each state node | TestOrder_DocumentOrder_ShouldSpecifyTheCorrectDocumentOrderForEachStateNode | ported | JS `dfs` walks `Object.keys(node.states)` (document order); Go uses `node.ChildStates()` (documented as document order) because `StateNode.States` is an unordered map. The `[key, order]` tuples become a local `order1KeyOrder` struct. |

## API gaps added
None (`apigap_order_1.go` not created).

## Ambiguities for review
- order.test.ts:58 `Object.keys(node.states)` -> `ChildStates()`; relies on the contract guarantee "returns child nodes in document order" (machine.go).
- order.test.ts:62 `machine.root` -> `machine.Root` (field on `StateMachine`; `RootNode()` also exists).
