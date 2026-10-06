// main.ts runs a demo actor (real 1 s timers, console output) at import time; its console output is
// muted and the process exits explicitly once the trace is written. The trace drives `workflow`
// with deterministic stubs: every promise actor settles after 80 ms (so a snapshot taken right after
// `send` still shows the invoking state) and 'Get status for book' answers from a fixed list.
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
  }
});
const request = {
  type: 'bookLendingRequest',
  book: { title: 'Dune', id: '7' },
  lender: { name: 'Jane Roe', address: '1 Main St', phone: '555-0100' }
};
const t = await runTrace(
  'workflow-book-lending-sleep-unresolved-delay',
  logic,
  [
    { send: request }, // -> 'Get Book Status'
    { wait: 120 }, // 'onloan' -> 'Report Status To Lender'
    { wait: 80 }, // -> 'Wait for Lender response'
    { send: { type: 'holdBook' } }, // -> 'Request Hold'
    { wait: 120 }, // onDone -> 'Sleep two weeks'; delay PT2W has no implementation, so the after event is queued at once -> 'Get Book Status'
    { wait: 80 }, // 'available' -> 'Check Out Book' / 'Checking out book'
    { wait: 80 }, // -> 'Notifying Lender'
    { wait: 80 } // -> 'Check Out Book' / 'End'
  ]
);
process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
process.exit(0);
