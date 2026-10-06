import { ticTacToeMachine } from '../../../references/xstate/examples/tic-tac-toe-react/src/ticTacToeMachine.ts';
import { runTrace } from '../trace.ts';

const t = await runTrace('draw', ticTacToeMachine, [
  { send: { type: 'PLAY', value: 0 } },
  { send: { type: 'PLAY', value: 1 } },
  { send: { type: 'PLAY', value: 2 } },
  { send: { type: 'PLAY', value: 4 } },
  { send: { type: 'PLAY', value: 3 } },
  { send: { type: 'PLAY', value: 5 } },
  { send: { type: 'PLAY', value: 7 } },
  { send: { type: 'PLAY', value: 6 } },
  { send: { type: 'PLAY', value: 8 } },
  { send: { type: 'PLAY', value: 0 } }, // ignored in gameOver.draw
  { send: { type: 'RESET' } },
  { send: { type: 'PLAY', value: 4 } },
]);
console.log(JSON.stringify(t, null, 2));
