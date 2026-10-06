// Runs the example's own entry (main.ts) with the real 500 ms `after` delay; its 10 hops end after about 5 s.
await import('../../../references/xstate/examples/workflow-filling-water/main.ts');
await new Promise((r) => setTimeout(r, 5800));
process.exit(0);
