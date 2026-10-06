// main.ts runs a demo actor at import time (real 1000 ms timer, prints to the
// console). Its console output is muted for the import; the trace runs the
// example's own greetingFunction (real 1000 ms) and writes the JSON with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-event-greeting/main.ts');
const { runTrace } = await import('../trace.ts');

const t = await runTrace('workflow-event-greeting', workflow, [
  { send: { type: 'unknown' } }, // no transition in Waiting
  { send: { type: 'greet', greet: { name: 'Jenny' } } }, // Waiting -> Greet (invokes greetingFunction)
  { send: { type: 'greet', greet: { name: 'Bob' } } }, // ignored in Greet
  { wait: 300 }, // greetingFunction (1000 ms) still pending: stays in Greet
  { wait: 1200 }, // promise resolved: onDone assigns greeting -> Greeted (final)
  { send: { type: 'greet', greet: { name: 'Ada' } } } // ignored once done
]);
process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
