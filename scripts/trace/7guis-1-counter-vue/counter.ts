import { counterMachine } from '../../../references/xstate/examples/7guis-1-counter-vue/src/counterMachine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('7guis-1-counter-vue', counterMachine, [
  { send: { type: 'increase' } },
  { send: { type: 'increase' } },
  { send: { type: 'increase' } },
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
