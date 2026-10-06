// The root machine never exposes its final state's `output` (xstate only reads the ROOT `output` for snapshot.output),
// so the winning-bid reduce is recorded by calling the machine's own BiddingEnded output function directly.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-car-auction-bids/main.ts');

const bid = (amount: number, id: string) => ({
  carid: 'car123',
  amount,
  bidder: { id, firstName: id, lastName: id.toUpperCase() }
});
const outputFn = (workflow as any).root.states.BiddingEnded.output as (a: { context: { bids: unknown[] } }) => unknown;

const cases = [
  { name: 'single bid', bids: [bid(3000, 'a')] },
  { name: 'increasing: later higher bid wins', bids: [bid(3000, 'a'), bid(4000, 'b')] },
  { name: 'decreasing: earlier higher bid wins', bids: [bid(4000, 'a'), bid(3000, 'b')] },
  { name: 'tie: the later bid wins', bids: [bid(4000, 'a'), bid(4000, 'b')] },
  { name: 'highest in the middle', bids: [bid(3000, 'a'), bid(5000, 'b'), bid(4000, 'c')] },
  { name: 'tie of the maximum among three', bids: [bid(5000, 'a'), bid(1000, 'b'), bid(5000, 'c')] },
  { name: 'no bids: reduce without initial value throws', bids: [] }
].map((c) => {
  try {
    return { ...c, output: outputFn({ context: { bids: c.bids } }) };
  } catch (e: any) {
    return { ...c, error: e.message };
  }
});
process.stdout.write(JSON.stringify({ name: 'winning-bid', cases }, null, 2) + '\n', () => process.exit(0));
