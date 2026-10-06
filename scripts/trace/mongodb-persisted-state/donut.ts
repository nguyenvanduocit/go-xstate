import { donutMachine } from '../../../references/xstate/examples/mongodb-persisted-state/donutMachine.ts';
import { runTrace } from '../trace.ts';
// Visits every state (ingredients, directions.makeDough, directions.mix with both
// regions mixing/mixed, directions.allMixed -> onDone, fry, flip, dry, glaze, serve)
// and the loop serve -> ingredients. Unknown / out-of-state events change nothing.
const t = await runTrace('donut', donutMachine, [
  { send: { type: 'ANOTHER_DONUT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'MIXED_DRY' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'MIXED_WET' } },
  { send: { type: 'MIXED_DRY' } },
  { send: { type: 'MIXED_WET' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'UNKNOWN' } },
  { send: { type: 'ANOTHER_DONUT' } }
]);
console.log(JSON.stringify(t, null, 2));
