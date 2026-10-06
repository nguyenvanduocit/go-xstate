import { getGameOutcome } from './lib/load.ts';

// getGameOutcome over hand-picked boards: each of the 8 lines for x, a win for
// o, a line found first when two lines are complete, a draw, and boards still
// in play. Golden: { name, cases: [ { board, outcome } ] }
const boards: (string | null)[][] = [];
const n = null;
const lines = [
  [0, 1, 2],
  [3, 4, 5],
  [6, 7, 8],
  [0, 3, 6],
  [1, 4, 7],
  [2, 5, 8],
  [0, 4, 8],
  [2, 4, 6]
];
for (const line of lines) {
  const b = Array(9).fill(null);
  for (const i of line) b[i] = 'x';
  boards.push(b);
}
boards.push(['o', 'o', 'o', n, 'x', n, 'x', n, 'x']); // o wins
boards.push(['x', 'x', 'x', 'x', 'o', 'o', 'x', 'o', 'o']); // two lines: [0,1,2] first
boards.push(['x', 'o', 'x', 'x', 'o', 'o', 'o', 'x', 'x']); // draw
boards.push([n, n, n, n, n, n, n, n, n]); // empty
boards.push(['x', 'o', 'x', n, 'o', n, n, 'x', n]); // in play

const cases = boards.map((board) => ({ board, outcome: getGameOutcome(board as any) }));
console.log(JSON.stringify({ name: 'store-tic-tac-toe/outcome', cases }, null, 2));
