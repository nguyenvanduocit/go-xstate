// Rehydration trace: the machine is persisted in the middle of the parallel `mix`
// state (mixDry done, mixWet still mixing), JSON round-tripped like a MongoDB document,
// and a new actor is created from it as main.ts does
// (`createActor(donutMachine, { state: restoredState?.persistedState })`).
// The persisted snapshot is recorded as the golden file's `input`.
import { createActor } from 'xstate';
import { donutMachine } from '../../../references/xstate/examples/mongodb-persisted-state/donutMachine.ts';
import { view } from '../trace.ts';

const first = createActor(donutMachine).start();
for (const type of ['NEXT', 'NEXT', 'MIXED_DRY']) first.send({ type });
const persisted = JSON.parse(JSON.stringify(first.getPersistedSnapshot()));
first.stop();

const actor = createActor(donutMachine, { state: persisted });
actor.start();
const steps: any[] = [{ step: 'start', snapshot: view(actor.getSnapshot()) }];
for (const event of [
  { type: 'MIXED_DRY' },
  { type: 'NEXT' },
  { type: 'MIXED_WET' },
  { type: 'NEXT' },
  { type: 'NEXT' },
  { type: 'NEXT' },
  { type: 'NEXT' },
  { type: 'ANOTHER_DONUT' }
]) {
  actor.send(event);
  steps.push({ step: { send: event }, snapshot: view(actor.getSnapshot()) });
}
actor.stop();
console.log(JSON.stringify({ name: 'donut-restored', input: persisted, steps }, null, 2));
