// main.ts runs a demo actor at import time (real 5000 ms timer, console output). The trace scripts
// only need the exported `workflow`, so console.log is muted for good and the JSON is written with
// fs.writeSync; the script exits explicitly because the demo actor is still running.
import { writeSync } from 'node:fs';
import { fromPromise } from 'xstate';

console.log = () => {};
const { workflow } = await import('../../../../references/xstate/examples/workflow-monitor-job/main.ts');

export const input = { job: { name: 'job1' } };

const after100 = <T>(value: T) =>
  new Promise<T>((resolve) => setTimeout(() => resolve(value), 100));

// Deterministic replacements of the four actors: each resolves after 100 ms.
// checkJobStatus answers with the given statuses in order (the last one repeats);
// `undefined` is a status that is neither SUCCEEDED nor FAILED.
export function stubbedWorkflow(statuses: Array<'SUCCEEDED' | 'FAILED' | undefined>) {
  let call = 0;
  return workflow.provide({
    actors: {
      submitJob: fromPromise(async () => after100({ jobuid: '123' })),
      checkJobStatus: fromPromise(async () =>
        after100({ jobStatus: statuses[Math.min(call++, statuses.length - 1)] })
      ),
      reportJobSucceeded: fromPromise(async () => after100(undefined)),
      reportJobFailed: fromPromise(async () => after100(undefined))
    }
  });
}

export function finish(json: unknown): never {
  writeSync(1, JSON.stringify(json, null, 2) + '\n');
  process.exit(0);
}
