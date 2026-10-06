import { fromPromise } from 'xstate';
import { friendMachine } from '../../../references/xstate/examples/friends-list-react/src/friendMachine.ts';
import { runTrace } from '../trace.ts';

// Deterministic stand-in for the 1000 ms network request of the real saveUser.
const machine = friendMachine.provide({
  actors: {
    saveUser: fromPromise(async () => {
      await new Promise((resolve) => setTimeout(resolve, 30));
      return true;
    })
  }
});

const input = { name: 'Ann' };
const t = await runTrace(
  'friend',
  machine,
  [
    { send: { type: 'SET_NAME', value: 'ignored' } }, // reading: SET_NAME not handled
    { send: { type: 'CANCEL' } }, // reading -> reading via root CANCEL
    { send: { type: 'EDIT' } }, // reading -> editing
    { send: { type: 'SET_NAME', value: 'Anna' } },
    { send: { type: 'EDIT' } }, // editing: EDIT not handled
    { send: { type: 'CANCEL' } }, // editing -> reading, name reverts to prevName
    { send: { type: 'EDIT' } },
    { send: { type: 'SET_NAME', value: 'Bob' } },
    { send: { type: 'SAVE' } }, // editing -> saving
    { send: { type: 'SET_NAME', value: 'ignored' } }, // saving: SET_NAME not handled
    { wait: 100 }, // invoke onDone: saving -> reading, prevName = name
    { send: { type: 'EDIT' } },
    { send: { type: 'SET_NAME', value: 'Carl' } },
    { send: { type: 'SAVE' } },
    { send: { type: 'CANCEL' } }, // saving -> reading, invoked promise is discarded
    { wait: 100 }, // nothing happens when the discarded promise settles
    { send: { type: 'SAVE' } }, // reading: SAVE not handled
    { send: { type: 'unknown' } }
  ],
  { input }
);
console.log(JSON.stringify({ name: t.name, clock: false, input, steps: t.steps }, null, 2));
