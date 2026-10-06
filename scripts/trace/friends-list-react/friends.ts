import { friendsMachine } from '../../../references/xstate/examples/friends-list-react/src/friendsMachine.ts';
import { runTrace } from '../trace.ts';

// friendsMachine builds actor ids with Math.random(); feed it fixed values.
// (0.1234567890123).toString(36).substring(7) === 'xkxayr', etc.
const randoms = [
  0.1234567890123, 0.2345678901234, 0.3456789012345, 0.4567890123456,
  0.5678901234567, 0.6789012345678
];
let next = 0;
Math.random = () => randoms[next++];

const add = (name: string) => ({ send: { type: 'FRIENDS.ADD', name } });
const change = (name: string) => ({ send: { type: 'NEW_FRIEND.CHANGE', name } });
const remove = (index: number) => ({ send: { type: 'FRIEND.REMOVE', index } });
const t = await runTrace('friends', friendsMachine, [
  change('Alice'),
  add(''), // guard false: empty
  add('   '), // guard false: whitespace only
  add('﻿'), // guard false: U+FEFF is JS whitespace
  add('Alice'), // guard true: spawns friend-xkxayr, newFriendName reset
  add('x'), // guard true on event.name, but newFriendName is '' (child gets name '')
  change('Bob'),
  add('\u0085'), // guard true: U+0085 is not JS whitespace
  add(' \t'), // guard false: NBSP and tab are JS whitespace
  remove(7), // index out of range: nothing removed
  remove(-1), // negative index: nothing removed
  remove(0.5), // non-integer index: nothing removed
  remove(1), // removes the middle friend
  remove(0),
  remove(0),
  remove(0), // empty list
  change('Zed'),
  add('Zed'), // fourth spawn, fresh id after removals
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify({ name: t.name, clock: false, input: null, steps: t.steps }, null, 2));
