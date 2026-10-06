import { counterMachine } from '../../../references/xstate/examples/counter/src/counterMachine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('counter', counterMachine, [
  { send: { type: 'increment' } },
  { send: { type: 'increment' } },
  { send: { type: 'decrement' } },
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
