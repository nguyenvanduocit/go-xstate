// main.ts runs a demo actor at import time and prints to the console. Its console
// output is muted for the import; the trace runs the example's own machine and
// writes the JSON with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-hello/main.ts');
const { runTrace } = await import('../trace.ts');

// The machine's initial state is final: it is done right at start, so the only
// observable behaviour besides the start snapshot is that events are ignored.
const t = await runTrace('workflow-hello', workflow, [
  { send: { type: 'unknown' } } // ignored once done
]);
process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
