// Runs the example's own entry (main.ts) with real timers (about 2 s: MakeAppointmentAction waits
// 2000 ms). main.ts stamps the appointment with `new Date().toISOString()`, so Date is pinned to a
// fixed instant for the run to make the recorded text deterministic.
const RealDate = Date;
class FixedDate extends RealDate {
  constructor(...args: any[]) {
    if (args.length === 0) super('2026-01-02T03:04:05.678Z');
    else super(...(args as [any]));
  }
}
globalThis.Date = FixedDate as DateConstructor;

await import('../../../references/xstate/examples/workflow-event-based-service/main.ts');
