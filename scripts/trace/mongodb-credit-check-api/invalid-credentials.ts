// Verifying Credentials onError: bad SSN, empty firstName, non-string lastName; each returns to Entering Information with an ErrorMessage.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('invalid-credentials', creditCheckMachine, [
    submit('123'),
    { advance: 20 },
    submit('123456789', ''),
    { advance: 20 },
    submit('123456789', 'Gavin', 123),
    { advance: 20 },
    { send: { type: 'unknown' } }
  ])
);
