// Runs the example's own entry (main.ts) with the real 1 s actors; its 3-actor chain ends after about 3 s.
await import('../../../references/xstate/examples/workflow-book-lending/main.ts');
await new Promise((r) => setTimeout(r, 3500));
process.exit(0);
