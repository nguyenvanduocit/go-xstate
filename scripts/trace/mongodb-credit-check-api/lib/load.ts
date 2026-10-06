// Loads the JS example's own machine.ts with its services module replaced by
// lib/stub-services.ts, and routes console.log into `logs` so stdout stays JSON.
import { plugin } from 'bun';
import { resolve } from 'node:path';
import { format } from 'node:util';

const here = import.meta.dir;

export const logs: string[] = [];
console.log = (...a: unknown[]) => void logs.push(format(...a));

// An actor that errors with no error handler is reported through setTimeout(() => { throw }) by xstate.
process.on('uncaughtException', (e) => process.stderr.write('uncaught (expected for unhandled actor errors): ' + e + '\n'));

plugin({
  name: 'stub-services',
  setup(build) {
    build.onLoad({ filter: /mongodb-credit-check-api\/services\/machineLogicService\.ts$/ }, async () => ({
      contents: await Bun.file(resolve(here, 'stub-services.ts')).text(),
      loader: 'ts'
    }));
  }
});

export const { creditCheckMachine } = await import('../../../../references/xstate/examples/mongodb-credit-check-api/machine.ts');

export const submit = (SSN: string, firstName: any = 'Gavin', lastName: any = 'Bauman') => ({
  send: { type: 'Submit', SSN, lastName, firstName }
});

export function emit(trace: unknown) {
  process.stdout.write(JSON.stringify(trace, null, 2) + '\n');
}
