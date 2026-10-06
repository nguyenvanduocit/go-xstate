// main.ts runs a demo actor at import time (real 1000 ms timer, prints to the
// console). Its console output is muted for the import; the trace itself uses a
// deterministic sendEmail stub (100 ms) and writes the JSON with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-async-function/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const logic = workflow.provide({
  actors: {
    sendEmail: fromPromise(async () => {
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
    })
  }
});

const input = { customer: 'david@example.com' };
const t = await runTrace(
  'workflow-async-function',
  logic,
  [
    { wait: 30 }, // sendEmail still pending: stays in 'Send email'
    { wait: 150 } // promise resolved: onDone -> 'Email sent' (final), output undefined
  ],
  { input }
);
process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
