// Drives friendsMachine together with the friendMachine actors it spawns, using the
// REAL saveUser (1000 ms). Golden format (replayed by friends_children_test.go):
//   { name, steps: [ { step: 'start' | {send, to?} | {wait}, snapshot, actors } ] }
//   `to` is an index into the root's context.friends at that moment: the event goes to
//   that friend instead of the root. `actors` is the view() of every friend actor seen so
//   far, keyed by id (a removed friend stays listed, showing its final snapshot).
import { friendsMachine } from '../../../references/xstate/examples/friends-list-react/src/friendsMachine.ts';
import { createActor } from 'xstate';
import { view } from '../trace.ts';

let next = 0;
const randoms = [0.1234567890123, 0.2345678901234, 0.3456789012345];
Math.random = () => randoms[next++];

type Step = { send: Record<string, unknown>; to?: number } | { wait: number };
const toFriend = (to: number, type: string, value?: string) => ({
  send: value === undefined ? { type } : { type, value },
  to
});
const steps: Step[] = [
  { send: { type: 'NEW_FRIEND.CHANGE', name: 'Dana' } },
  { send: { type: 'FRIENDS.ADD', name: 'Dana' } }, // friend 0 gets input.name 'Dana'
  toFriend(0, 'EDIT'),
  toFriend(0, 'SET_NAME', 'Dani'),
  toFriend(0, 'SAVE'),
  { wait: 1100 }, // real saveUser (1000 ms) resolves: saving -> reading, prevName = 'Dani'
  toFriend(0, 'EDIT'),
  toFriend(0, 'SET_NAME', 'Dee'),
  toFriend(0, 'CANCEL'), // editing -> reading, name back to 'Dani'
  { send: { type: 'FRIENDS.ADD', name: 'x' } }, // friend 1 gets input.name '' (context.newFriendName)
  toFriend(1, 'EDIT'),
  toFriend(1, 'SET_NAME', 'Eve'),
  toFriend(1, 'SAVE'), // friend 1 is saving when it is removed
  { send: { type: 'FRIEND.REMOVE', index: 1 } }, // stopChild: friend 1 stops, saving never completes
  { wait: 1100 },
  toFriend(0, 'EDIT'), // friend 0 still works
  { send: { type: 'FRIEND.REMOVE', index: 0 } },
  { wait: 50 }
];

const actor = createActor(friendsMachine);
const seen = new Map<string, any>();
const record = () => {
  const snap = actor.getSnapshot();
  for (const ref of snap.context.friends) seen.set(ref.id, ref);
  const actors: Record<string, unknown> = {};
  for (const id of [...seen.keys()].sort()) actors[id] = view(seen.get(id).getSnapshot());
  return { snapshot: view(snap), actors };
};
const out: any[] = [];
actor.start();
out.push({ step: 'start', ...record() });
for (const step of steps) {
  if ('wait' in step) await new Promise((r) => setTimeout(r, step.wait));
  else if (step.to !== undefined) actor.getSnapshot().context.friends[step.to].send(step.send as any);
  else actor.send(step.send as any);
  out.push({ step, ...record() });
}
actor.stop();
console.log(JSON.stringify({ name: 'friends-children', steps: out }, null, 2));
