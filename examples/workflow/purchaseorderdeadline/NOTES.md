# Purchase order deadline

Ports `references/xstate/examples/workflow-purchase-order-deadline/main.ts` with
the original 15-second root deadline. Creating and confirming an order do not
reset the deadline. Shipment completes the order; expiry invokes CancelOrder,
which waits one second before reaching OrderCancelled. Its unhandled rejection
errors the machine. OrderFinishedEvent is declared upstream but has no handler.

The success trace covers all event transitions, completion before deadline,
and ignored events after completion. Three deadline traces expire in each active
waiting state, checking 14999ms versus 15000ms. The cancel-error trace records the
unhandled invocation error. All use the original JS machine, a SimulatedClock,
and a deterministic 100ms cancellation service, matched by Go tests.

The stdout fixture runs the original entry with timer durations scaled by 1/100.
It confirms an order at 10 seconds, cancels it at 15 seconds, and ignores shipment
at 20 seconds. Run uses original durations; RunWith permits the same scale as the
fixture. The real Go CancelOrder service is exercised in the stdout test.

The loader strips only the demo actor for machine traces and resolves Cockatiel
from the pinned dependency in the patient-onboarding trace directory. Run
`bun install` in `scripts/trace/workflow-new-patient-onboarding/lib/deps` before
regenerating on a fresh checkout. The upstream retry policy is constructed but
never used by CancelOrder, so no retry is added. TypeScript, package-manager, and
bundler configuration are excluded. Stopping a Go actor cancels its pending
wait; abandoned JS promises may continue logging after stop.
