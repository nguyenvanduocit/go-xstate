// main.ts runs a demo actor at import time (real 1000 ms timers, prints to the
// console). Its console output is muted for the import; the trace uses a
// deterministic batchMathFunction stub (50 ms, same result mapping) and writes
// the JSON with process.stdout. Four expressions.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-math-problem/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const logic = workflow.provide({
  actors: {
    batchMathFunction: fromPromise(async ({ input }: { input: { problems: string[] } }) => {
      await new Promise<void>((resolve) => setTimeout(resolve, 50));
      return input.problems.map((problem) => ({ problem, result: `Solved ${problem}` }));
    })
  }
});

const input = { expressions: ['2+2', '4-1', '10x3', '20/2'] };
const t = await runTrace(
  'workflow-math-problem',
  logic,
  [
    { wait: 20 }, // batch pending: stays in 'Solve'
    { wait: 100 } // resolved: onDone assigns results -> 'Solved' (final)
  ],
  { input }
);
process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
