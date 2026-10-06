// Submit with valid credentials and no failure: every region fetches (existing-report guard false), all three succeed, middle score, rate, RatesProvided.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('happy', creditCheckMachine, [
    submit('123456789'),
    submit('123456789'), // Submit is ignored while verifying
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 100 },
    { advance: 40 },
    { advance: 60 },
    submit('123456789'), // ignored in the final state
    { send: { type: 'unknown' } }
  ])
);
