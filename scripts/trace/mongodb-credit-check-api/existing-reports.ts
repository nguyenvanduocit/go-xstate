// Gavperian bureau call rejects; the retry hits the existing-report guard for EquiGavin and GavUnion (stored reports override the context scores) and fetches Gavperian.
import { runTrace } from './lib/record.ts';
import { creditCheckMachine, emit, submit } from './lib/load.ts';

// Advance to each service boundary explicitly; observation does not advance time.
emit(
  await runTrace('existing-reports', creditCheckMachine, [
    submit('999999998'),
    { advance: 20 },
    { advance: 40 },
    { advance: 100 },
    { advance: 100 },
    { advance: 100 },
    submit('888888888'),
    { advance: 20 },
    { advance: 40 },
    { advance: 300 },
    { advance: 40 },
    { advance: 60 },
    { advance: 60 }
  ])
);
