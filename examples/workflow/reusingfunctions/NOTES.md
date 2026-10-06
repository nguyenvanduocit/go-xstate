# Reusing functions

Ports both machines in `references/xstate/examples/workflow-reusing-functions/main.ts`:
the payment-confirmation child and its parent. The parent forwards payment events,
observes snapshots, and receives the child's ConfirmationCompletedEvent and done
event. It remains active after the child completes. The accountId in the event is
never assigned upstream; the port preserves null when passing it to checkfunds.
Available funds are determined by payment amount < 1000, replacing the event's
initial funds value. Both email services receive the customer as applicant.

The two success traces cover 999 versus 1000, all child states, forwarding,
snapshot observation, wildcard events, sendParent, completion, and an ignored
second payment to the finished child. ConfirmBasedOnFunds is transient; the trace
records its selected email state. Three error traces reject each service and
record parent, child, and parent-received events. These traces execute the
original machines, with only timers or failing external service actors replaced.
Go tests compare both snapshots and the event list after every step.

The stdout fixture runs the original entry. Its console boundary projects raw
MachineSnapshot logs to the shared portable view: status, value, context, output,
tags, and child IDs. This excludes JavaScript runtime internals (machine/function
objects, actor references, and inspection methods) that have no Go counterpart.
Service and event logs are unchanged. Run prints the same projection; RunWith
accepts a timer duration. Run coordinates service-start logs with snapshot logs
to preserve JavaScript's synchronous pre-await output order across Go goroutines.
The parent has no final state, so Run returns after the
child's done event and stops the parent instead of waiting for parent completion.

The trace loader exposes the unexported parent without changing its configuration
and strips only the demo entry. TypeScript/package-manager/bundler configuration
is excluded. Go cancellation stops service timers; abandoned JS promises may
continue logging after actor stop.
