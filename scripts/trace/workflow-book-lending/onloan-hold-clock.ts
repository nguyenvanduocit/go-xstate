// main.ts runs a demo actor (real 1 s timers, console output) at import time; its console output is
// muted and the process exits explicitly once the trace is written. The trace drives `workflow`
// with deterministic stubs: every promise actor settles after 80 ms (so a snapshot taken right after
// `send` still shows the invoking state) and 'Get status for book' answers from a fixed list.
// The SimulatedClock drives only the `after` delay (PT2W is given 1000 ms here); actor settling uses real time.
// Observation steps: the first `wait` after an event is 120 ms, each following one 80 ms, so every
// `wait` lands 40 ms after exactly one actor hop (no drift between hops).
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-book-lending/main.ts');
const { fromPromise } = await import('xstate');
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const { runTrace } = await import('../trace.ts');

const statuses: string[] = ['onloan', 'available'];
let n = 0;
const stub = fromPromise(async () => {
  await sleep(80);
});
const logic = workflow.provide({
  actors: {
    'Get status for book': fromPromise(async () => {
      await sleep(80);
      return { status: statuses[n++] };
    }),
    'Send status to lender': stub,
    'Request hold for lender': stub,
    'Cancel hold request for lender': stub,
    'Check out book with id': stub,
    'Notify Lender for checkout': stub
  },
  delays: { PT2W: 1000 }
});
const request = {
  type: 'bookLendingRequest',
  book: { title: 'Dune', id: '7' },
  lender: { name: 'Jane Roe', address: '1 Main St', phone: '555-0100' }
};
const t = await runTrace(
  'workflow-book-lending-onloan-hold-clock',
  logic,
  [
    { send: { type: 'holdBook' } }, // ignored in 'Book Lending Request'
    { send: request }, // -> 'Get Book Status' (book.status 'unknown' until the actor answers)
    { send: { type: 'holdBook' } }, // ignored in 'Get Book Status'
    { wait: 120 }, // 'onloan' -> 'Book Status Decision' always[0] -> 'Report Status To Lender'
    { wait: 80 }, // onDone -> 'Wait for Lender response'
    { send: request }, // ignored there
    { send: { type: 'holdBook' } }, // -> 'Request Hold'
    { wait: 120 }, // onDone -> 'Sleep two weeks'
    { advance: 999 }, // PT2W (1000 ms here) not elapsed yet
    { advance: 1 }, // after PT2W -> 'Get Book Status' again
    { wait: 120 }, // 'available' -> always[1] -> 'Check Out Book' / 'Checking out book'
    { wait: 80 }, // onDone -> 'Notifying Lender'
    { wait: 80 }, // onDone -> 'Check Out Book' / 'End' (final child; the parent has no onDone, so the machine stays active)
    { send: { type: 'holdBook' } } // ignored
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n');
process.exit(0);
