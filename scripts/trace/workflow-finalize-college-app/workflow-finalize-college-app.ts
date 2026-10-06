// main.ts runs a demo actor at import time (real 1000 ms timers, prints to the
// console). Its console output is muted for the import; the trace itself uses a
// deterministic finalizeApplicationFunction stub (100 ms) and writes the JSON
// with process.stdout.
const log = console.log;
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-finalize-college-app/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const logic = workflow.provide({
  actors: {
    finalizeApplicationFunction: fromPromise(async ({ input }: { input: { applicantId: string } }) => {
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
      return { applicantId: input.applicantId };
    })
  }
});

const input = { applicantId: '123' };
const t = await runTrace(
  'workflow-finalize-college-app',
  logic,
  [
    { send: { type: 'unknown' } }, // unhandled event: no change
    { send: { type: 'SATScoresReceived' } }, // always guard false (1 of 3)
    { send: { type: 'SATScoresReceived' } }, // repeated event: still false
    { send: { type: 'ApplicationSubmitted' } }, // guard false (2 of 3)
    { send: { type: 'RecommendationLetterReceived' } }, // guard true -> FinalizingApplication (invoke)
    { send: { type: 'unknown' } }, // ignored while finalizing
    { wait: 30 }, // finalize still pending
    { wait: 150 } // onDone -> Finalized (final), output undefined
  ],
  { input }
);
process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
