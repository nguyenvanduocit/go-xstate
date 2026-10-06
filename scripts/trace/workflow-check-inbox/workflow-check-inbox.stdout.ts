// Runs the example's own entry (main.ts) with real timers. The workflow never completes (the
// schedule actor reminds every 2000 ms forever), so the process is ended at 4500 ms: one full cycle
// (reminder at 2000, inbox read at 3000, texts sent at 3100 and 3500) and the next reminder at 4000,
// whose inbox read would finish at 5000.
await import('../../../references/xstate/examples/workflow-check-inbox/main.ts');
setTimeout(() => process.exit(0), 4500);
