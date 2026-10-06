// Parent workflow `checkcarvitals` with the vitals sub-workflow's four checks
// replaced by deterministic real-timer stubs (20/40/60/80 ms, distinct values).
// main.ts runs a demo actor at import time (about 7 s, real timers); its console
// output is muted and the trace is written with process.stdout.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-car-vitals/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const check = (ms: number, value: number) =>
  fromPromise(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, ms));
    return { value };
  });

const vitals = (workflow as any).implementations.actors.vitalscheck.provide({
  actors: {
    checkCoolantLevel: check(20, 90),
    checkTirePressure: check(40, 32),
    checkBattery: check(60, 12),
    checkOilPressure: check(80, 45)
  }
});
const logic = workflow.provide({ actors: { vitalscheck: vitals } });

const t = await runTrace(
  'workflow-car-vitals',
  logic,
  [
    { send: { type: 'CarTurnedOffEvent' } }, // off while already WhenCarIsOn
    { send: { type: 'UnknownEvent' } }, // unhandled
    { send: { type: 'CarTurnedOnEvent' } }, // -> DoCarVitalChecks, vitals child running
    { send: { type: 'CarTurnedOnEvent' } }, // ignored in DoCarVitalChecks
    { wait: 150 }, // all four checks done, vitals final -> onDone -> CheckContinueVitalChecks
    { advance: 999 }, // after 1000 not yet elapsed
    { advance: 1 }, // after 1000 -> DoCarVitalChecks, fresh vitals child
    { wait: 30 }, // checks still pending
    { send: { type: 'CarTurnedOffEvent' } }, // off during checks: child stopped
    { wait: 150 }, // stale promises settle, ignored
    { send: { type: 'CarTurnedOnEvent' } },
    { wait: 150 }, // -> CheckContinueVitalChecks
    { send: { type: 'CarTurnedOffEvent' } }, // off while waiting: after timer cancelled
    { advance: 1000 } // nothing fires
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n');
