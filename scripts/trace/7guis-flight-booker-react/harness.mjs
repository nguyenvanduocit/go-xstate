// Loads the example's flightMachine.ts under two shims, without editing the example:
//  1. '@xstate/react' -> inert virtual module (React is not installed in the trace runner; the trace never uses the React context).
//  2. `new Date()` is pinned to FIXED_NOW while the module graph loads, so utils'
//     TODAY / TOMORROW constants are deterministic (TODAY = 2024-03-10).
// Not a trace script: gen.sh only runs *.ts files in this directory.
import { plugin } from 'bun';

plugin({
  name: 'xstate-react-stub',
  setup(build) {
    build.module('@xstate/react', () => ({
      exports: { createActorContext: () => ({}) },
      loader: 'object'
    }));
  }
});

const FIXED_NOW = Date.parse('2024-03-10T12:00:00Z');

export async function loadFlightBookerMachine() {
  const RealDate = Date;
  class FixedDate extends RealDate {
    constructor(...args) {
      if (args.length === 0) super(FIXED_NOW);
      else super(...args);
    }
    static now() {
      return FIXED_NOW;
    }
  }
  globalThis.Date = FixedDate;
  try {
    const mod = await import(
      '../../xstate/examples/7guis-flight-booker-react/src/machines/flightMachine.ts'
    );
    return mod.flightBookerMachine;
  } finally {
    globalThis.Date = RealDate;
  }
}
