// Deterministic drive of tilesMachine. Math.random (shuffleTiles) returns a fixed sequence, one value per
// call; running out is an error, so a `shuffle` that must be ignored (inside `playing`) is proven to not shuffle.
//
// Fisher-Yates runs i = 15..1 with j = floor(r * (i + 1)). r = 0.99 gives j = i (no change) for every i,
// and r = 0 at i = 1 gives j = 0, so IDENTITY is 15 x 0.99 and SWAP01 (swap tiles 0 and 1) is 14 x 0.99 then 0.
import { tilesMachine } from '../../../references/xstate/examples/tiles/src/tilesMachine.ts';
import { runTrace } from '../trace.ts';

const IDENTITY = Array.from({ length: 15 }, () => 0.99);
const SWAP01 = [...Array.from({ length: 14 }, () => 0.99), 0];
const randoms = [...SWAP01, ...IDENTITY, ...SWAP01];
let next = 0;
Math.random = () => {
  if (next >= randoms.length) throw new Error('random sequence exhausted');
  return randoms[next++];
};

const tile = (index: number) => ({ index, x: index % 4, y: Math.floor(index / 4) });
const select = (i: number) => ({ send: { type: 'tile.select', tile: tile(i) } });
const hover = (i: number) => ({ send: { type: 'tile.hover', tile: tile(i) } });
const move = { send: { type: 'tile.move' } };
const cancel = { send: { type: 'move.canceled' } };
const shuffle = { send: { type: 'shuffle' } };

const t = await runTrace('tiles', tilesMachine, [
  // start: nothing but shuffle is handled
  { send: { type: 'unknown' } },
  select(5),
  move,
  cancel,
  hover(1),
  // root `shuffle` (SWAP01): start -> playing.selecting, tiles = [1,0,2,...]; `always` guard is false
  shuffle,
  // playing.selecting: shuffle is swallowed by playing; hover/move/cancel have no handler
  shuffle,
  hover(2),
  move,
  cancel,
  // selecting -> selected
  select(5),
  shuffle,
  select(6), // no handler in `selected`
  // move without a hover: isAdjacent false (hovered undefined) -> selecting, selected stays
  move,
  // far diagonal hover: not adjacent
  select(5),
  hover(10),
  move,
  // same row but two apart (5 and 7), same column but two apart (5 and 13): not adjacent
  select(5),
  hover(7),
  move,
  select(5),
  hover(13),
  move,
  // hover moves around, then cancel clears selected and hovered
  select(5),
  hover(6),
  hover(9),
  cancel,
  // adjacent moves: horizontal (5,6), vertical (5,9), then undo both
  select(5),
  hover(6),
  move,
  select(5),
  hover(9),
  move,
  select(5),
  hover(9),
  move,
  select(5),
  hover(6),
  move,
  // final adjacent swap (1,0) puts all tiles in order -> playing `always` -> gameOver
  select(1),
  hover(0),
  move,
  // gameOver: only shuffle is handled
  select(5),
  move,
  cancel,
  // shuffle (IDENTITY) from gameOver: playing entered, `always` returns to gameOver at once
  shuffle,
  // shuffle (SWAP01) from gameOver -> playing.selecting
  shuffle,
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
