// Shared trace format: the JS reference run and the Go example test must
// produce the same JSON for the same script.
//
// golden file = { name, steps: [ { step, snapshot } ] }
//   step: { send: Event } | { advance: ms } (SimulatedClock) | { wait: ms } (real time)
//   snapshot: { status, value, context, output, tags, children, error? }
import { createActor, SimulatedClock } from 'xstate';

export type Step =
  | { send: Record<string, unknown> }
  | { advance: number }
  | { wait: number };

export function view(snap: any) {
  const out: any = {
    status: snap.status,
    value: snap.value,
    context: snap.context ?? null,
    output: snap.output ?? null,
    tags: [...(snap.tags ?? [])].sort(),
    children: Object.keys(snap.children ?? {}).sort()
  };
  if (snap.error !== undefined) {
    out.error = snap.error instanceof Error ? snap.error.message : snap.error;
  }
  return JSON.parse(JSON.stringify(out));
}

export async function runTrace(
  name: string,
  logic: any,
  steps: Step[],
  options: { input?: unknown; useClock?: boolean } = {}
) {
  const clock = options.useClock ? new SimulatedClock() : undefined;
  const actor = createActor(logic, { input: options.input, clock });
  const out: any[] = [{ step: 'start', snapshot: null }];
  actor.start();
  out[0].snapshot = view(actor.getSnapshot());
  for (const step of steps) {
    if ('send' in step) actor.send(step.send as any);
    else if ('advance' in step) clock!.increment(step.advance);
    else await new Promise((r) => setTimeout(r, step.wait));
    out.push({ step, snapshot: view(actor.getSnapshot()) });
  }
  actor.stop();
  return { name, steps: out };
}
