// Runs the example's own entry (main.ts) and feeds it the stdin line `Submit`
// (process.stdin 'data' event) so the real actors print their messages.
await import('../../../references/xstate/examples/workflow-applicant-request/main.ts');
process.stdin.emit('data', Buffer.from('Submit\n'));
await new Promise((r) => setTimeout(r, 1300));
process.exit(0);
