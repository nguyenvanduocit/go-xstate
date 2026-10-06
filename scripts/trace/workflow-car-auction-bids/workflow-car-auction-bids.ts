// main.ts runs the whole demo at import time (about 2 s of real timers) and exports `workflow`.
// Console output is muted for that import; the trace uses the unmodified exported machine with a SimulatedClock
// for `after: BiddingDelay` (3000 ms, started at machine start, never reset by bids).
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-car-auction-bids/main.ts');
const { runTrace } = await import('../trace.ts');

const bid = (carid: string, amount: number, id: string, firstName: string, lastName: string) => ({
  type: 'CarBidEvent',
  bid: { carid, amount, bidder: { id, firstName, lastName } }
});

const t = await runTrace(
  'workflow-car-auction-bids',
  workflow,
  [
    { send: { type: 'unknown' } }, // ignored
    { advance: 1000 },
    { send: bid('car123', 3000, 'xyz', 'John', 'Wayne') },
    { advance: 1000 },
    { send: bid('car123', 4000, 'abc', 'Jane', 'Doe') },
    { send: bid('car123', 2500, 'low', 'Lo', 'Ball') }, // lower than the best: reduce keeps prev
    { send: bid('car123', 4000, 'tie', 'Tie', 'Breaker') }, // equal amount: reduce picks current
    { advance: 999 }, // 2999 ms: BiddingDelay not due
    { advance: 1 }, // 3000 ms: BiddingDelay fires -> BiddingEnded (final), output
    { send: bid('car123', 9000, 'late', 'Late', 'Comer') }, // done: ignored
    { advance: 5000 }
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n', () => process.exit(0));
