// main.ts runs the whole demo at import time (about 14 s of real timers) and exports `workflow`.
// Console output is muted for that import; the trace uses the unmodified exported machine
// (real produceReport, whose delay(1_000) never rejects: errorProbability is 0) with a SimulatedClock
// for `after: PT1H` and a real 1100 ms wait for the 1000 ms promise.
console.log = () => {}; // stays muted: the real produceReport logs too; the JSON goes out through process.stdout.write
const { workflow } = await import('../../../references/xstate/examples/workflow-accumulate-room-readings/main.ts');
const { runTrace } = await import('../trace.ts');

const reading = (kind: 'TemperatureEvent' | 'HumidityEvent', value: number) =>
  kind === 'TemperatureEvent'
    ? { type: kind, roomId: 'kitchen', temperature: value }
    : { type: kind, roomId: 'kitchen', humidity: value };

const t = await runTrace(
  'workflow-accumulate-room-readings',
  workflow,
  [
    { send: { type: 'unknown' } }, // ignored
    { send: reading('TemperatureEvent', 20) },
    { send: reading('TemperatureEvent', 25) }, // overwrites
    { advance: 5000 }, // PT1H not due
    { send: reading('HumidityEvent', 50) },
    { advance: 4999 }, // 9999 ms: still ConsumeReading
    { advance: 1 }, // PT1H fires, guard true -> GenerateReport (invokes produceReport)
    { send: reading('TemperatureEvent', 99) }, // ignored while generating
    { advance: 500 },
    { wait: 1100 }, // produceReport resolves -> onDone -> ConsumeReading, entry resets context
    { send: reading('HumidityEvent', 30) },
    { advance: 10000 }, // PT1H fires, guard false (temperature null): stays, timer is spent
    { send: reading('TemperatureEvent', 10) },
    { advance: 10000 } // nothing re-arms the timer
  ],
  { useClock: true }
);
process.stdout.write(JSON.stringify({ ...t, clock: true }, null, 2) + '\n', () => process.exit(0));
