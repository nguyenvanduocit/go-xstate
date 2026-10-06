import { toggleMachine } from '../../../references/xstate/examples/toggle/src/toggleMachine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('toggle', toggleMachine, [
  { send: { type: 'toggle' } },
  { send: { type: 'toggle' } },
  { send: { type: 'toggle' } },
  { send: { type: 'unknown' } },
  { send: { type: 'toggle' } }
]);
console.log(JSON.stringify(t, null, 2));
