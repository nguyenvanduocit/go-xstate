// Drives the real handlers of examples/server/workflow/index.ts through the
// fake router in lib/fake-express.ts and records every response.
import { loadServer } from './lib/fake-express.ts';

// Math.random() values; generateActorId() turns each into a 6-char id.
const srv = await loadServer([0.123456789, 0.987654321]);

const steps: any[] = [];
function call(method: 'GET' | 'POST', path: string, body?: unknown) {
  const request: any = { method, path };
  if (body !== undefined) request.body = body;
  steps.push({ request, response: srv.request(method, path, body) });
}

// Workflow ids returned by the two POST /workflows calls below.
const A = (0.123456789).toString(36).substring(2, 8);
const B = (0.987654321).toString(36).substring(2, 8);

call('GET', '/');
call('POST', '/workflows');
call('GET', `/workflows/${A}`);
call('POST', `/workflows/${A}`, { type: 'TIMER' });
call('GET', `/workflows/${A}`);
call('POST', `/workflows/${A}`, { type: 'TIMER' });
call('GET', `/workflows/${A}`);
call('POST', `/workflows/${A}`, { type: 'UNKNOWN' });
call('GET', `/workflows/${A}`);
call('POST', `/workflows/${A}`, { type: 'TIMER' });
call('GET', `/workflows/${A}`);
call('POST', `/workflows/${A}`, { type: 'TIMER', extra: 1 });
call('GET', `/workflows/${A}`);
call('POST', '/workflows');
call('GET', `/workflows/${B}`);
call('POST', `/workflows/${B}`, { type: 'TIMER' });
call('GET', `/workflows/${B}`);
call('GET', `/workflows/${A}`);
call('GET', '/workflows/missing');
call('POST', '/workflows/missing', { type: 'TIMER' });

srv.restoreRandom();
console.log(JSON.stringify({ name: 'express-workflow-server', steps }, null, 2));
