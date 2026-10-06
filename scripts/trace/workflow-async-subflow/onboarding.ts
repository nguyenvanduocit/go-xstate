import { runTrace } from '../trace.ts';
import { onboarding, promptStub, answersFor, finish } from './lib/load.ts';

// The onboarding machine on its own, so its states and context are visible in the trace.
const logic = onboarding.provide({ actors: { prompt: promptStub(answersFor('Ada')) } });

const t = await runTrace('onboarding', logic, [
  { send: { type: 'unknown' } }, // no transition in Welcome
  { wait: 20 }, // still Welcome, prompt pending
  { wait: 50 }, // Welcome -> Personalize, name assigned
  { wait: 60 }, // Personalize -> Completed (final)
  { send: { type: 'unknown' } } // ignored once done
]);
finish({ ...t, clock: false });
