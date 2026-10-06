// Drives the real handlers of index.ts (with real actorService.ts and machine.ts) and records
// every response. Persisted snapshots are read through GET after each real-time wait.
import { loadServer } from './lib/load-server.ts';

// Math.random values; generateActorId() turns each into a 6-char id.
const srv = await loadServer([0.123456789, 0.987654321]);
const A = (0.123456789).toString(36).substring(2, 8);
const B = (0.987654321).toString(36).substring(2, 8);

const steps: any[] = [];
async function call(method: 'GET' | 'POST', path: string, body?: unknown) {
  const request: any = { method, path };
  if (body !== undefined) request.body = body;
  steps.push({ request, response: await srv.request(method, path, body) });
}
async function wait(ms: number) {
  await new Promise((r) => setTimeout(r, ms));
  steps.push({ wait: ms });
}
const submit = (SSN: string, firstName = 'Gavin', lastName = 'Bauman') => ({ type: 'Submit', SSN, lastName, firstName });

await call('GET', '/');
await call('POST', '/workflows');
await call('GET', `/workflows/${A}`);

// invalid credentials: back to Entering Information with an ErrorMessage
await call('POST', `/workflows/${A}`, submit('123'));
await call('GET', `/workflows/${A}`);
await wait(60);
await call('GET', `/workflows/${A}`);

// valid credentials: same timeline as the happy trace (stub events at 20, 60, 160, 260, 360, 400, 460 ms)
await call('POST', `/workflows/${A}`, submit('123456789'));
await call('GET', `/workflows/${A}`);
for (const ms of [40, 70, 100, 100, 70, 50, 60]) {
  await wait(ms);
  await call('GET', `/workflows/${A}`);
}
await call('POST', `/workflows/${A}`, { type: 'unknown' });
await call('GET', `/workflows/${A}`);

// unknown workflow ids
await call('GET', '/workflows/missing');
await call('POST', '/workflows/missing', submit('123456789'));

// a second, independent workflow
await call('POST', '/workflows');
await call('POST', `/workflows/${B}`, { type: 'unknown' });
await call('GET', `/workflows/${B}`);
await call('GET', `/workflows/${A}`);

srv.restoreRandom();
process.stdout.write(JSON.stringify({ name: 'mongodb-credit-check-api-server', steps }, null, 2) + '\n');
process.exit(0);
