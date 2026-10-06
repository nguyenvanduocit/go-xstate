// checkReportsTable rejects for all three bureaus: every region reaches FetchingFailed, allSucceeded false.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('reports-table-error', creditCheckMachine, [
    submit('777777777'),
    { advance: 20 },
    { advance: 40 },
    { send: { type: 'unknown' } }
  ])
);
