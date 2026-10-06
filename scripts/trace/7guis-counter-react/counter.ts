import { counterMachine } from '../../../references/xstate/examples/7guis-counter-react/src/counterMachine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('7guis-counter-react', counterMachine, [
  { send: { type: 'INCREMENT' } },
  { send: { type: 'INCREMENT' } },
  { send: { type: 'INCREMENT' } },
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
