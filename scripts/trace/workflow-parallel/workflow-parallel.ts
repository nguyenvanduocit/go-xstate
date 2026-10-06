// main.ts runs a demo actor at import time (real 1000 ms / 3000 ms timers, prints to the
// console). Its console output is muted for the import; the trace itself uses deterministic
// shortDelay (100 ms) and longDelay (300 ms) stubs and writes the JSON with process.stdout.
// The process exits explicitly because the demo actor's 3 s timer would keep it alive.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-parallel/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const delay = (ms: number) =>
  fromPromise(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, ms));
  });
const logic = workflow.provide({
  actors: { shortDelay: delay(100), longDelay: delay(300) }
});

const t = await runTrace('workflow-parallel', logic, [
  { wait: 30 }, // both branches active, both invokes pending
  { send: { type: 'unknown' } }, // no transition handles it
  { wait: 120 }, // t~150: shortDelay done -> ShortDelayBranch.done (final); LongDelayBranch still active
  { wait: 250 }, // t~400: longDelay done -> both regions final -> ParallelExec onDone -> Success (final)
  { send: { type: 'unknown' } } // ignored once done
]);
process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
process.exit(0);
