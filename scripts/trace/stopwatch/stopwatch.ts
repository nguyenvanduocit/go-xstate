import { fromCallback } from 'xstate';
import { stopwatchMachine } from '../../../references/xstate/examples/stopwatch/src/stopwatchMachine.ts';
import { runTrace } from '../trace.ts';

// The real `ticks` actor uses wall-clock setInterval(10). The trace swaps in a
// no-op callback actor and sends TICK explicitly, which reaches the same
// `running.on.TICK` transition deterministically.
const machine = stopwatchMachine.provide({
  actors: { ticks: fromCallback(() => () => {}) }
});

const t = await runTrace('stopwatch', machine, [
  { send: { type: 'stop' } }, // ignored in stopped
  { send: { type: 'TICK' } }, // ignored in stopped
  { send: { type: 'start' } },
  { send: { type: 'start' } }, // ignored in running
  { send: { type: 'TICK' } },
  { send: { type: 'TICK' } },
  { send: { type: 'TICK' } },
  { send: { type: 'stop' } }, // elapsed kept
  { send: { type: 'TICK' } }, // ignored again
  { send: { type: 'start' } }, // resumes from 3
  { send: { type: 'TICK' } },
  { send: { type: 'reset' } }, // from running: elapsed 0, back to stopped
  { send: { type: 'reset' } }, // from stopped
  { send: { type: 'start' } },
  { send: { type: 'TICK' } },
  { send: { type: 'stop' } },
  { send: { type: 'reset' } }, // from stopped with elapsed 1
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
