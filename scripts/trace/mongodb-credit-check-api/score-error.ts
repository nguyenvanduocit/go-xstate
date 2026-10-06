// determineMiddleScore rejects and DeterminingMiddleScore has no onError: the machine ends in status error.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('score-error', creditCheckMachine, [
    submit('555555555'),
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 100 },
    { advance: 40 }
  ])
);
