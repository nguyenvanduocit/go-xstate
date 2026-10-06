import { donutMachine } from '../../../references/xstate/examples/persisted-donut-maker/donutMachine.ts';
import { runTrace } from '../trace.ts';
const t = await runTrace('persisted-donut-maker', donutMachine, [
  // ingredients -> directions.makeDough
  { send: { type: 'NEXT' } },
  { send: { type: 'ANOTHER_DONUT' } }, // not handled in directions
  // makeDough -> mix (parallel)
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } }, // not handled in mix
  { send: { type: 'MIXED_DRY' } },
  { send: { type: 'MIXED_DRY' } }, // already mixed: no-op
  // mixWet done -> mix done -> allMixed (final) -> directions done -> fry
  { send: { type: 'MIXED_WET' } },
  { send: { type: 'NEXT' } }, // fry -> flip
  { send: { type: 'NEXT' } }, // flip -> dry
  { send: { type: 'NEXT' } }, // dry -> glaze
  { send: { type: 'NEXT' } }, // glaze -> serve
  { send: { type: 'unknown' } },
  { send: { type: 'ANOTHER_DONUT' } }, // serve -> ingredients
  // second donut, regions finish in the other order
  { send: { type: 'NEXT' } },
  { send: { type: 'NEXT' } },
  { send: { type: 'MIXED_WET' } },
  { send: { type: 'MIXED_DRY' } }
]);
console.log(JSON.stringify(t, null, 2));
