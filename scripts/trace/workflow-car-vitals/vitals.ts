// The (non-exported) `vitalscheck` sub-workflow of main.ts, run on its own with
// deterministic real-timer stubs: coolant 100 ms, tire 200 ms, battery 300 ms,
// oil 400 ms. Covers both branches of the `always` guard.
console.log = () => {};
const { workflow } = await import('../../../references/xstate/examples/workflow-car-vitals/main.ts');
const { fromPromise } = await import('xstate');
const { runTrace } = await import('../trace.ts');

const check = (ms: number, value: number) =>
  fromPromise(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, ms));
    return { value };
  });

const logic = (workflow as any).implementations.actors.vitalscheck.provide({
  actors: {
    checkCoolantLevel: check(100, 90),
    checkTirePressure: check(200, 32),
    checkBattery: check(300, 12),
    checkOilPressure: check(400, 45)
  }
});

const t = await runTrace('vitals', logic, [
  { wait: 150 }, // coolant only: guard false
  { wait: 100 }, // + tire (t=250)
  { wait: 100 }, // + battery (t=350)
  { wait: 100 } // + oil (t=450): guard true -> VitalsChecked (final)
]);
process.stdout.write(JSON.stringify({ ...t, clock: false }, null, 2) + '\n');
