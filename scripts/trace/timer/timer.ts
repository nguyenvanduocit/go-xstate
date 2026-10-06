import { fromCallback } from 'xstate';
import { timerMachine } from '../../../references/xstate/examples/timer/src/timerMachine.ts';
import { runTrace } from '../trace.ts';

// The real `ticks` actor uses wall-clock setInterval(1000). The trace swaps in a
// no-op callback actor and sends TICK explicitly, which reaches the same
// `running.on.TICK` transition deterministically.
const machine = timerMachine.provide({
  actors: { ticks: fromCallback(() => () => {}) }
});

const t = await runTrace('timer', machine, [
  { send: { type: 'start' } }, // guard seconds > 0 fails: stays stopped
  { send: { type: 'reset' } }, // root guard fails: no-op
  { send: { type: 'stop' } }, // ignored in stopped
  { send: { type: 'TICK' } }, // ignored in stopped
  { send: { type: 'second' } },
  { send: { type: 'second' } },
  { send: { type: 'minute' } }, // 62
  { send: { type: 'reset' } }, // root guard passes: 0
  { send: { type: 'second' } }, // 1
  { send: { type: 'start' } }, // running
  { send: { type: 'start' } }, // ignored in running
  { send: { type: 'minute' } }, // ignored in running
  { send: { type: 'second' } }, // ignored in running
  { send: { type: 'TICK' } }, // 0 -> always -> stopped
  { send: { type: 'minute' } }, // 60
  { send: { type: 'start' } },
  { send: { type: 'TICK' } }, // 59
  { send: { type: 'stop' } }, // seconds kept
  { send: { type: 'TICK' } }, // ignored again
  { send: { type: 'start' } }, // resumes from 59
  { send: { type: 'reset' } }, // from running: 0 then always -> stopped
  { send: { type: 'reset' } }, // from stopped with 0: guard fails
  { send: { type: 'second' } },
  { send: { type: 'second' } },
  { send: { type: 'start' } },
  { send: { type: 'TICK' } },
  { send: { type: 'TICK' } }, // 0 -> stopped
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
