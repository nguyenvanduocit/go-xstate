// main.ts runs a demo actor (real 500 ms timers, console output) at import time; its console output is
// muted and the process exits explicitly once the trace is written. The trace drives `workflow` with a
// SimulatedClock so the 500 ms `after` delay is deterministic.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-filling-water/main.ts');
const { runTrace } = await import('../trace.ts');

const input = { current: 0, max: 2 };
const t = await runTrace(
  'workflow-filling-water-fill',
  workflow,
  [
    { advance: 499 }, // AddWater: delay not elapsed yet
    { advance: 1 }, // after 500: current 1 -> CheckIfFull always[0] (1 < 2) -> AddWater
    { advance: 500 }, // current 2 -> CheckIfFull always[1] (2 < 2 false) -> GlassFull (final)
    { advance: 500 } // done: nothing happens
  ],
  { input, useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true, input }, null, 2) + '\n');
process.exit(0);
