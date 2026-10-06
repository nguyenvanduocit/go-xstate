// Steps shared by the store trace. Takes every branch of the `played` handler
// (taken cell, out-of-range position, move after the game ended), a win for x,
// a win for o, a full board with no winning line, and `reset` (from a finished
// and from a fresh game).
const play = (position: number) => ({ send: { type: 'played', position } });

export const steps = [
  play(0), // x
  play(0), // taken cell: ignored
  play(9), // out of range: ignored
  play(-1), // out of range: ignored
  play(3), // o
  play(1), // x
  play(4), // o
  play(2), // x completes the top row
  play(8), // game is over: ignored
  { send: { type: 'reset' } },
  { send: { type: 'reset' } }, // already initial
  play(0), // x
  play(3), // o
  play(1), // x
  play(4), // o
  play(8), // x
  play(5), // o completes the middle row
  { send: { type: 'unknown' } }, // no handler
  { send: { type: 'reset' } },
  play(0), // x
  play(1), // o
  play(2), // x
  play(4), // o
  play(3), // x
  play(5), // o
  play(7), // x
  play(6), // o
  play(8), // x fills the board, no line
  play(0), // game is over: ignored
  { send: { type: 'reset' } }
];
