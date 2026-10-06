# Provision orders

Ports [`workflow-provision-orders/main.ts`](../../../references/xstate/examples/workflow-provision-orders/main.ts):
validate an order's ID, item, and quantity in that order, then apply the order or
invoke the matching exception workflow. Nested exception completion leads to
the machine's final state.

The JavaScript fixtures in `testdata/` cover success, each missing-field branch,
and a rejection that matches none of the error guards. The recorders in
`scripts/trace/workflow-provision-orders/` shorten the upstream one-second actor
delays to 50 ms; Go trace tests use the same delay. `TestStdout` checks the demo's
missing-ID output against a recording. `TestRunOutput` also checks output and
validation order for the other branches with shorter injected delays.

The upstream services log and wait; no external order service is contacted.
Rejections from the apply-order and exception-handler actors are not covered.
The context keeps all order fields as strings, including quantity, to match the
upstream input contract. There is no browser UI to port.

Run from `examples/`:

```sh
go test -race -count=1 ./workflow/provisionorders
```
