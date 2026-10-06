// EquiGavin and GavUnion bureau calls reject (FetchingFailed, allSucceeded false, ErrorMessage); retry with stored Gavperian score hits the existing-report guard.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('bureau-failure', creditCheckMachine, [
    submit('999999999'),
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 100 },
    submit('888888888'),
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 40 },
    { advance: 60 }
  ])
);
