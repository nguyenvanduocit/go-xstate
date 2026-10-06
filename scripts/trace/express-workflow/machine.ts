import { machine } from '../../../references/xstate/examples/express-workflow/machine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('express-workflow-machine', machine, [
  { send: { type: 'TIMER' } },
  { send: { type: 'TIMER' } },
  { send: { type: 'TIMER' } },
  { send: { type: 'UNKNOWN' } },
  { send: { type: 'TIMER' } },
  { send: { type: 'TIMER' } },
  { send: { type: 'TIMER' } }
]);
console.log(JSON.stringify(t, null, 2));
