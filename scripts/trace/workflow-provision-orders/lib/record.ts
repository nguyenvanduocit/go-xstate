// Shared by the trace scripts of workflow-provision-orders.
//
// main.ts runs a demo actor at import time and its actors wait 1000 ms with
// setTimeout. Console output is muted and setTimeout(…, 1000) is shortened to
// 50 ms BEFORE main.ts is imported, so the trace drives the example's own
// actors (real validation logic) quickly and deterministically. The caller
// writes the JSON and exits.
const realSetTimeout = globalThis.setTimeout;
(globalThis as any).setTimeout = (fn: any, ms?: number, ...args: any[]) =>
  realSetTimeout(fn, ms === 1000 ? 50 : ms, ...args);
console.log = () => {};

export const { workflow } = await import(
  '../../../../references/xstate/examples/workflow-provision-orders/main.ts'
);
export const { runTrace } = await import('../../trace.ts');

// Timeline (ms): every actor settles 50 ms after it starts. Snapshots are read
// at 25 ms (provisionOrderFunction pending), 75 ms (second actor pending),
// 150 ms (workflow done).
export const steps = [
  { send: { type: 'Unknown' } }, // ignored while provisionOrderFunction is pending
  { wait: 25 },
  { wait: 50 },
  { wait: 75 }
];

export async function record(
  name: string,
  logic: any,
  input: unknown,
  stepList: any[] = steps
) {
  const t = await runTrace(name, logic, stepList, { input });
  process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
  process.exit(0);
}
