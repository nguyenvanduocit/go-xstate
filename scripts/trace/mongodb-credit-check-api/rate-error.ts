// generateInterestRates rejects and FetchingRates has no onError: the machine ends in status error.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('rate-error', creditCheckMachine, [
    submit('666666666'),
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 100 },
    { advance: 40 },
    { advance: 60 },
    submit('666666666')
  ])
);
