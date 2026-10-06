// Runs the example's own entry (main.ts) with real timers (about 14 s, then its last timer ends the process at 21 s).
// The only random source, delay()'s error probability, is 0 for every call, so the output is deterministic.
await import('../../../references/xstate/examples/workflow-accumulate-room-readings/main.ts');
