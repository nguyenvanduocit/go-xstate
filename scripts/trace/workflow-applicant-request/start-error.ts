// main.ts runs a demo actor and a stdin listener at import time; its console output is muted and
// the process exits explicitly once the trace is written. The trace drives `workflow` with
// deterministic stubs (50 ms, so a snapshot right after `send` still shows the invoking state) for the two 1 s promise actors (the real ones are covered by the stdout trace).
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-applicant-request/main.ts');
const { fromPromise } = await import('xstate');
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const { runTrace } = await import('../trace.ts');

const logic = workflow.provide({
  actors: {
    startApplicationWorkflowId: fromPromise(async () => {
      await sleep(50);
      throw new Error('boom');
    }),
    sendRejectionEmailFunction: fromPromise(async () => {
      await sleep(50);
    })
  }
});
const input = { applicant: { fname: 'John', lname: 'Stockton', age: 30, email: 'js@something.com' } };
const t = await runTrace(
  'workflow-applicant-request-start-error',
  logic,
  [
    { send: { type: 'Submit' } }, // guard true -> StartApplication
    { wait: 75 }, // onError -> RejectApplication
    { wait: 75 } // onDone -> End
  ],
  { input }
);
process.stdout.write(JSON.stringify({ ...t, input }, null, 2) + '\n');
process.exit(0);
