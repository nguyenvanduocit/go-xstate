// BiddingEnded's output reduces context.bids without an initial value: with no bids it throws a TypeError.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-car-auction-bids/main.ts');
const { runTrace } = await import('../trace.ts');

const t = await runTrace(
  'no-bids',
  workflow,
  [
    { advance: 2999 },
    { advance: 1 } // BiddingDelay fires with an empty bids list
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n', () => process.exit(0));
