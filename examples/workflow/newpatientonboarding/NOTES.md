# New patient onboarding

Ports `references/xstate/examples/workflow-new-patient-onboarding/main.ts`:
Idle, all three invoked services, the nested final state, and End. Success clears
patient; each service error finishes with patient retained. Services retry only
`ServiceNotAvailable`, with an initial attempt plus ten retries and 3-second backoff.
Each attempt waits one second and fails with probability 0.5 by default.

The success trace visits the three service states and completes. Three failure
traces cover every onError exit. Traces also send ignored events before and after
completion. The nested Done state is transient and appears through its resulting
End snapshot. Traces substitute deterministic promise services (200ms); Go uses
the same boundary stubs. They execute the original machine configuration.

The three stdout fixtures execute the original entry and services: immediate
success, one retry then success, and retry exhaustion. Only Math.random and timer
speed are replaced. Go exercises the real Services and Retry implementations
against these fixtures. Additional tests check nonretryable errors and backoff.
Run uses the original random behavior; RunWith accepts deterministic boundaries.
The writer formats objects like Bun, which records the reference output.

The loader omits only the demo actor when obtaining the original exported machine.
Cockatiel 3.2.1 is pinned under
`scripts/trace/workflow-new-patient-onboarding/lib/deps`; run `bun install` there
before regenerating fixtures on a fresh checkout. Both workflow harnesses use it.
No Go dependency is required. TypeScript/package-manager/bundler configuration is
not ported. Cancellation of a Go actor cancels its pending waits; abandoned JS
promises may continue logging, which is outside the entry's completion behavior.
