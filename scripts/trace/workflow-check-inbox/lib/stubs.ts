// Deterministic replacements for the actors of workflow-check-inbox/main.ts, shared by the traces.
// main.ts runs a demo actor at import time; its console output is muted for the import.
console.log = () => {};
const { workflow } = await import('../../../../references/xstate/examples/workflow-check-inbox/main.ts');
const { fromCallback, fromPromise } = await import('xstate');
const { runTrace } = await import('../../trace.ts');

const delay = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

// checkInboxFunction: main.ts waits 1000 ms; here 100 ms. Same two messages.
export const checkInboxFunction = fromPromise(async () => {
  await delay(100);
  return [
    { subject: 'Hello', priority: 'high' },
    { subject: 'Hi', priority: 'low' }
  ];
});

// sendTextsFunction: main.ts waits 100 ms (high) / 500 ms (low) per message; here 20 / 100 ms.
export const sendTextsFunction = fromPromise(
  async ({ input }: { input: { messages: { priority: string }[] } }) => {
    await Promise.all(
      input.messages.map((m) => delay(m.priority === 'high' ? 20 : 100))
    );
    return { status: 'success' };
  }
);

// schedule: main.ts reminds every input.interval (2000) ms; `interval` overrides it, 0 means never.
export const schedule = (interval: number) =>
  fromCallback<any, { interval: number }>(({ sendBack }) => {
    if (interval === 0) return () => {};
    const i = setInterval(() => sendBack({ type: 'reminder' }), interval);
    return () => clearInterval(i);
  });

export { workflow, runTrace };

export async function emit(name: string, interval: number, steps: any[]) {
  const logic = workflow.provide({
    actors: { schedule: schedule(interval), checkInboxFunction, sendTextsFunction }
  });
  const t = await runTrace(name, logic, steps);
  process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
  process.exit(0);
}
