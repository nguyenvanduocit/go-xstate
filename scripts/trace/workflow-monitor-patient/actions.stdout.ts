// Records what the three actions print when the machine is driven by hand (no
// timer, no randomness): one event of each kind, an unknown event, and a second
// patient to show the context is read at action time.
const realSetInterval = globalThis.setInterval;
(globalThis as any).setInterval = () => 0;
const realLog = console.log;
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-monitor-patient/main.ts');
console.log = realLog;
(globalThis as any).setInterval = realSetInterval;
const { createActor } = await import('xstate');

const vital = (type: string, patientId: string) => ({
  type,
  source: 'monitoringSource',
  id: 'event1',
  time: '2024-01-01T00:00:00.000Z',
  patientId,
  data: { value: 'v' }
});

const actor = createActor(workflow, { input: { patientId: 'patient42' } });
actor.start();
actor.send(vital('org.monitor.highRespirationRate', 'patient42') as any);
actor.send(vital('org.monitor.highBodyTemp', 'patient42') as any);
actor.send({ type: 'unknown' } as any);
actor.send(vital('org.monitor.highBloodPressure', 'patient42') as any);
actor.stop();
process.exit(0);
