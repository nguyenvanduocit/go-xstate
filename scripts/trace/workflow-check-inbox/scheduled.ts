// The schedule actor drives the workflow: reminder every 300 ms (main.ts: 2000 ms).
// Timeline: reminder 300, inbox read 400, texts sent 500 (high 20 ms, low 100 ms), reminder 600.
import { emit } from './lib/stubs.ts';

await emit('workflow-check-inbox/scheduled', 300, [
  { wait: 250 }, // 250: before the first reminder, Idle
  { wait: 100 }, // 350: CheckInbox
  { wait: 100 }, // 450: SendTextForHighPriority
  { wait: 100 }, // 550: Idle, messages kept
  { wait: 100 } // 650: CheckInbox again (second reminder at 600)
]);
