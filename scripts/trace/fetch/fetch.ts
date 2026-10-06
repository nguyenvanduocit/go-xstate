// src/fetchMachine.ts imports getGreeting from src/index.ts, and index.ts runs a demo actor at
// import time, so index.ts must load first (the other order hits a TDZ error on the cycle).
// Its console output is muted and Math.random pinned for that import; the trace itself uses
// the provided deterministic fetchUser below and writes the JSON with process.stdout.
Math.random = () => 0.9;
console.log = () => {};
await import('../../../references/xstate/examples/fetch/src/index.ts');
const { fetchMachine } = await import('../../../references/xstate/examples/fetch/src/fetchMachine.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

// Attempts 1 and 2 reject, attempt 3 resolves.
let attempts = 0;
const logic = fetchMachine.provide({
  actors: {
    fetchUser: fromPromise(async ({ input }: { input: { name: string } }) => {
      attempts++;
      if (attempts < 3) throw new Error('boom');
      return { greeting: `Hello, ${input.name}!` };
    })
  }
});

const t = await runTrace(
  'fetch',
  logic,
  [
    { send: { type: 'RETRY' } }, // ignored in idle
    { advance: 1000 }, // idle has no timers
    { send: { type: 'FETCH' } }, // idle -> loading (attempt 1)
    { wait: 50 }, // onError -> failure (arms the 1000ms after)
    { advance: 500 }, // still failure
    { send: { type: 'FETCH' } }, // ignored in failure
    { send: { type: 'RETRY' } }, // failure -> loading (attempt 2); the old after timer is cancelled
    { wait: 50 }, // onError -> failure (new after timer, due 1000ms later)
    { advance: 500 }, // 500ms after failure re-entered, 1000ms after the first entry: a non-cancelled first timer would fire here; still failure
    { advance: 499 }, // 999ms after failure was re-entered: still failure
    { advance: 1 }, // 1000ms after re-entry: after -> loading (attempt 3)
    { wait: 50 }, // onDone -> success, data assigned
    { send: { type: 'FETCH' } }, // ignored in success
    { send: { type: 'RETRY' } }, // ignored in success
    { advance: 5000 } // success is terminal for timers
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n');
