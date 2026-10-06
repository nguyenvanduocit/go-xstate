// Runs the example's own entry (src/index.ts), which prints every snapshot.
// Math.random is pinned to 0.9 so getGreeting resolves (its only source of nondeterminism).
Math.random = () => 0.9;
await import('../../../references/xstate/examples/fetch/src/index.ts');
