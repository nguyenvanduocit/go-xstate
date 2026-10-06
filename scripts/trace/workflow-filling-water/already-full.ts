// main.ts runs a demo actor (real 500 ms timers, console output) at import time; its console output is
// muted and the process exits explicitly once the trace is written. Input with current > max: the
// initial CheckIfFull takes always[1] at once, AddWater is never entered.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-filling-water/main.ts');
const { runTrace } = await import('../trace.ts');

const input = { current: 7, max: 5 };
const t = await runTrace(
  'workflow-filling-water-already-full',
  workflow,
  [
    { advance: 500 } // already done: nothing happens
  ],
  { input, useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true, input }, null, 2) + '\n');
process.exit(0);
