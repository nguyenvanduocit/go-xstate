// main.ts runs a demo actor at import time: it opens a readline interface on stdin and writes the
// first question to process.stdout. The trace scripts only need the exported `workflow`, so the
// import happens with process.stdout.write muted; the traces use deterministic prompt stubs
// (below) and write their JSON with fs.writeSync.
import { writeSync } from 'node:fs';
import { fromPromise } from 'xstate';

const realWrite = process.stdout.write.bind(process.stdout);
process.stdout.write = (() => true) as typeof process.stdout.write;
const { workflow } = await import('../../../../references/xstate/examples/workflow-async-subflow/main.ts');
process.stdout.write = realWrite;

// The `onboarding` machine is not exported by main.ts; it is registered as an actor of `workflow`.
export const onboarding = (workflow as any).implementations.actors.onboarding;

// Deterministic replacement of the `prompt` actor: answers from a table after 50 ms and fails on a
// question that is not in the table, so the exact question text is part of the trace.
export const promptStub = (answers: Record<string, string>) =>
  fromPromise(async ({ input }: { input: { question: string } }) => {
    await new Promise<void>((resolve) => setTimeout(resolve, 50));
    if (!(input.question in answers)) throw new Error(`unexpected question: ${input.question}`);
    return { response: answers[input.question] };
  });

export const answersFor = (name: string) => ({
  'What is your name?': name,
  [`Welcome ${name}, press enter to finish the onboarding process`]: ''
});

// The main.ts demo actor keeps stdin open, so the script exits explicitly after writing.
export function finish(json: unknown): never {
  writeSync(1, JSON.stringify(json, null, 2) + '\n');
  process.exit(0);
}

export { workflow };
