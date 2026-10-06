// main.ts runs a demo actor at import time (real 2000 ms timer, prints to the console). The trace
// scripts only need the exported `workflow`, so the import happens with console.log muted; the
// traces use a deterministic MakeAppointmentAction stub (below) and write their JSON with
// fs.writeSync.
import { writeSync } from 'node:fs';
import { fromPromise } from 'xstate';

const realLog = console.log;
console.log = () => {};
const { workflow } = await import('../../../../references/xstate/examples/workflow-event-based-service/main.ts');
console.log = realLog;

// Deterministic replacement of MakeAppointmentAction: resolves after 100 ms with the same shape as
// the real actor, a fixed date, and an appointmentId that echoes the patient name so the trace
// shows that `input.patientInfo` reached the actor.
export const FIXED_DATE = '2026-01-02T03:04:05.678Z';
export const appointmentStub = fromPromise(
  async ({ input }: { input: { patientInfo: { name: string } } }) => {
    await new Promise<void>((resolve) => setTimeout(resolve, 100));
    return {
      appointmentInfo: {
        appointmentId: `1234:${input.patientInfo.name}`,
        appointmentDate: FIXED_DATE
      }
    };
  }
);

// The main.ts demo actor keeps its 2000 ms timer alive, so the script exits explicitly after writing.
export function finish(json: unknown): never {
  writeSync(1, JSON.stringify(json, null, 2) + '\n');
  process.exit(0);
}

export { workflow };
