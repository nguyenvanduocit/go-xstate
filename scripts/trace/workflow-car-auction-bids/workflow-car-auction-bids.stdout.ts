// Runs the example's own entry (main.ts) with real timers (about 3 s: bids at 1 s and 2 s, BiddingDelay fires at 3 s).
// The output is deterministic: no random or wall-clock value is printed.
await import('../../../references/xstate/examples/workflow-car-auction-bids/main.ts');
