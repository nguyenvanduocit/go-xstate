// Deterministic drive of snakeMachine. Two sources of nondeterminism are pinned:
//  - Math.random (apple placement) returns the fixed sequence below, one value per call;
//    running out of values is an error, so the Go side must consume exactly the same calls.
//  - the `ticks` actor (setInterval 80ms, real time) is replaced by a no-op callback and
//    TICK events are sent by the script instead.
import { fromCallback } from 'xstate';
import { snakeMachine } from '../../../references/xstate/examples/snake-react/src/snakeMachine.ts';
import { runTrace } from '../trace.ts';

const randoms = [
  0.73, 0.48, // (18,7): on the snake -> rejected, retry
  0.77, 0.48, // (19,7)
  0.13, 0.21, // (3,3)
  0.22, 0.7 //   (5,10)
];
let next = 0;
Math.random = () => {
  if (next >= randoms.length) throw new Error('random sequence exhausted');
  return randoms[next++];
};

const machine = snakeMachine.provide({
  actors: { ticks: fromCallback(() => {}) }
});

const tick = { send: { type: 'TICK' } };
const key = (dir: string) => ({ send: { type: 'ARROW_KEY', dir } });
const ticks = (n: number) => Array.from({ length: n }, () => tick);

const t = await runTrace('snake-react/snake', machine, [
  // New Game: events other than ARROW_KEY are ignored
  tick,
  { send: { type: 'NEW_GAME' } },
  // New Game -> Moving (dir Right, move snake on entry)
  key('Right'),
  // Moving: TICK moves (head 14..17), no guard fires
  ...ticks(4),
  // head reaches the apple at (18,7): grow, score 1, apple placement retries once (random 1,2 on snake)
  tick,
  // NEW_GAME is ignored while Moving
  { send: { type: 'NEW_GAME' } },
  // eat second apple at (19,7): score 2, apple -> (3,3)
  tick,
  // ARROW_KEY keeps Moving (target is the state itself, no re-entry): it only saves dir, the
  // snake moves on the next TICK
  key('Down'),
  tick,
  key('Left'),
  tick,
  // opposite of the saved dir (Left) is ignored by 'save dir'
  key('Right'),
  // turn Up and tick: the head lands on a body cell -> 'hit tail' -> Game Over
  key('Up'),
  tick,
  // Game Over ignores TICK, ARROW_KEY
  tick,
  key('Left'),
  // reset keeps highScore
  { send: { type: 'NEW_GAME' } },
  // opposite of the initial Right is ignored by 'save dir', but New Game -> Moving still happens (entry moves Right)
  key('Left'),
  // eat the initial apple (18,7): score 1 < highScore 2; apple -> (5,10)
  ...ticks(5),
  // turn Up and run into the top wall: guard 'hit tail' false, 'hit wall' true (y 6..0, then -1)
  key('Up'),
  ...ticks(8),
  // Game Over again, then a final reset
  { send: { type: 'NEW_GAME' } }
]);
console.log(JSON.stringify(t, null, 2));
