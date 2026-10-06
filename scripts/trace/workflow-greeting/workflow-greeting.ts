// main.ts runs a demo actor at import time (real 1000 ms timer, prints to the
// console). Its console output is muted for the import; the trace itself uses a
// deterministic greetingFunction stub (100 ms) and writes the JSON with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-greeting/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const logic = workflow.provide({
  actors: {
    greetingFunction: fromPromise(async ({ input }: { input: { name: string } }) => {
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
      return { greeting: `Hello, ${input.name}!` };
    })
  }
});

const input = { person: { name: 'Jenny' } };
const t = await runTrace(
  'workflow-greeting',
  logic,
  [
    { wait: 30 }, // greetingFunction still pending: stays in 'Greet'
    { wait: 150 } // resolved: onDone assigns greeting -> 'Greeted' (final), root output undefined
  ],
  { input }
);
process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
