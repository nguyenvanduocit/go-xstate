// Runs the example's own entry (main.ts) with the real 1000 ms actors: the
// hardcoded order has an empty id, so the MissingId exception path prints.
await import('../../../references/xstate/examples/workflow-provision-orders/main.ts');
await new Promise((r) => setTimeout(r, 2500));
process.exit(0);
