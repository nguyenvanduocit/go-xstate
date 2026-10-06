import { ticTacToeMachine } from '../../../references/xstate/examples/tic-tac-toe-react/src/ticTacToeMachine.ts';
import { runTrace } from '../trace.ts';

const t = await runTrace('x-wins', ticTacToeMachine, [
  { send: { type: 'RESET' } }, // ignored while playing
  { send: { type: 'unknown' } }, // ignored
  { send: { type: 'PLAY', value: 0 } },
  { send: { type: 'PLAY', value: 0 } }, // occupied cell: guard isValidMove false
  { send: { type: 'PLAY', value: 9 } }, // out of range: guard false
  { send: { type: 'PLAY', value: -1 } }, // out of range: guard false
  { send: { type: 'PLAY', value: 3 } },
  { send: { type: 'PLAY', value: 1 } },
  { send: { type: 'PLAY', value: 4 } },
  { send: { type: 'PLAY', value: 2 } },
  { send: { type: 'PLAY', value: 5 } }, // ignored in gameOver
  { send: { type: 'RESET' } }, // gameOver -> playing, resetGame
  { send: { type: 'PLAY', value: 8 } } // fresh board after reset
]);
console.log(JSON.stringify(t, null, 2));
