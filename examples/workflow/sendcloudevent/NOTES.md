# Send CloudEvent on workflow completion

Ports `references/xstate/examples/workflow-send-cloudevent/main.ts`. Provisioning
runs all order timers concurrently and preserves input order in the results.
Context omits provisionedOrders until fulfillment, including an empty result
array for an empty input. The final state's output is preserved, but the root
has no output mapping: its output remains undefined. The upstream example does
not send an actual CloudEvent, so no network integration is added.

Traces cover multiple orders, empty orders, and an unhandled provisioning error,
plus ignored events after completion. The nonempty trace executes the original
service with its timers reduced to 100ms. The empty trace substitutes a promise
that returns [] after 100ms on both sides, to make its initial pending snapshot
deterministic: JavaScript schedules Promise.all([]) through a microtask, while a
Go promise may finish before Start returns. A separate empty-service stdout
fixture executes the original service without a stub and tests the actual Go
service's empty result. The error trace injects a rejecting service boundary.

The entry stdout fixture runs the original entry with accelerated timers and is
compared to RunWith. Run retains the one-second timers. The trace loader strips
only the entry when importing the exported machine. TypeScript/package-manager
and bundler configuration are excluded. Go cancellation stops pending timers;
JS promise timers may continue after an actor is stopped.
