// Runs the example's own entry (main.ts) with real timers (about 1 s: visaApprovedEvent
// is sent immediately, handleApprovedVisaWorkflowID takes 1000 ms). The output is deterministic.
await import('../../../references/xstate/examples/workflow-event-based/main.ts');
