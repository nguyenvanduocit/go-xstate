// main.ts runs a demo actor and a never-ending setInterval at import time. For the
// import, console.log is muted and setInterval is a no-op; the trace then drives
// the example's own `workflow` machine with the three vital-sign events.
const realLog = console.log;
console.log = () => {};
const realSetInterval = globalThis.setInterval;
(globalThis as any).setInterval = () => 0;
const { workflow } = await import('../../../references/xstate/examples/workflow-monitor-patient/main.ts');
(globalThis as any).setInterval = realSetInterval;
const { runTrace } = await import('../trace.ts');

const input = { patientId: 'patient1' };
const vital = (type: string, value: string) => ({
  type,
  source: 'monitoringSource',
  id: 'event1',
  time: '2024-01-01T00:00:00.000Z',
  patientId: 'patient1',
  data: { value }
});

const t = await runTrace(
  'workflow-monitor-patient',
  workflow,
  [
    { send: vital('org.monitor.highBodyTemp', '39.5') }, // sendTylenolOrder, stays in MonitorVitals
    { send: vital('org.monitor.highBloodPressure', '180/110') }, // callNurse
    { send: vital('org.monitor.highRespirationRate', '28') }, // callPulmonologist
    { send: { type: 'unknown' } }, // no transition
    { send: vital('org.monitor.highBodyTemp', '40.1') } // the machine never finishes
  ],
  { input }
);
console.log = realLog;
process.stdout.write(JSON.stringify({ ...t, clock: false, input }, null, 2) + '\n');
process.exit(0);
