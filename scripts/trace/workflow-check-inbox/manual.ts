// Manual reminders (the schedule actor never fires): every state, every transition, and the events
// ignored in CheckInbox / SendTextForHighPriority / Idle (unknown).
// Stubs: inbox read 100 ms, texts sent 20 ms (high) / 100 ms (low), so a full cycle ends 200 ms after the reminder.
import { emit } from './lib/stubs.ts';

await emit('workflow-check-inbox/manual', 0, [
  { send: { type: 'reminder' } }, // t=0: Idle -> CheckInbox
  { send: { type: 'reminder' } }, // ignored in CheckInbox
  { wait: 50 }, // t=50: inbox read still pending
  { wait: 100 }, // t=150: inbox read -> SendTextForHighPriority, messages assigned
  { send: { type: 'reminder' } }, // ignored in SendTextForHighPriority
  { wait: 100 }, // t=250: texts sent -> Idle, messages kept
  { send: { type: 'unknown' } }, // ignored in Idle
  { send: { type: 'reminder' } } // Idle -> CheckInbox again
]);
