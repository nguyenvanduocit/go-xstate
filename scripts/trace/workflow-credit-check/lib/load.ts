// main.ts runs a demo actor at import time (real 1000 ms timers, console output). The trace scripts
// only need the exported `workflow`, so console.log is muted for good and the JSON is written with
// fs.writeSync; the script exits explicitly because the demo actor is still running.
import { writeSync } from 'node:fs';
import { fromPromise } from 'xstate';

console.log = () => {};
const { workflow } = await import('../../../../references/xstate/examples/workflow-credit-check/main.ts');

export const customer = {
  id: 'customer123',
  name: 'John Doe',
  SSN: 123456,
  yearlyIncome: 50000,
  address: '123 MyLane, MyCity, MyCountry',
  employer: 'MyCompany'
};
export const input = { customer };

const after100 = <T>(value: T) =>
  new Promise<T>((resolve) => setTimeout(() => resolve(value), 100));

// Deterministic replacements of the three actors: each resolves after 100 ms.
export const creditCheckStub = (decision: string, score: number, reason: string) =>
  fromPromise(async () => after100({ id: 'customer123', score, decision, reason }));

export const startApplicationStub = fromPromise(async () =>
  after100({ application: { id: 'application123', status: 'Approved' } })
);

export const rejectionEmailStub = fromPromise(async () =>
  after100({ email: { id: 'email123', status: 'Sent' } })
);

// A credit check that never answers: only the 15 minute timeout can leave CheckCredit.
export const hangingCreditCheckStub = fromPromise(() => new Promise<never>(() => {}));

export function finish(json: unknown): never {
  writeSync(1, JSON.stringify(json, null, 2) + '\n');
  process.exit(0);
}

export { workflow };
