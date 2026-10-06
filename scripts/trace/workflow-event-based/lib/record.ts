// Shared by the trace scripts. main.ts runs a demo actor at import time (real 1000 ms
// timers, prints to the console), so its console output is muted for the import; the
// trace itself uses deterministic 100 ms stubs for the three promise actors and writes
// the JSON with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../../references/xstate/examples/workflow-event-based/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../../trace.ts');

const stub = () =>
  fromPromise(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, 100));
  });

const logic = workflow.provide({
  actors: {
    handleApprovedVisaWorkflowID: stub(),
    handleRejectedVisaWorkflowID: stub(),
    handleNoVisaDecisionWorkflowId: stub()
  }
});

export async function record(name: string, steps: Parameters<typeof runTrace>[2]) {
  const t = await runTrace(name, logic, steps, { useClock: true });
  process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n');
}
