import { runTrace } from '../trace.ts';
import { onboarding, promptStub, answersFor, finish } from './lib/load.ts';

// The user presses enter at the first question: name is assigned the empty string (not undefined),
// and the second question is built from it.
const logic = onboarding.provide({ actors: { prompt: promptStub(answersFor('')) } });

const t = await runTrace('onboarding-empty-name', logic, [
  { wait: 70 }, // Welcome -> Personalize, name ""
  { wait: 60 } // Personalize -> Completed
]);
finish({ ...t, clock: false });
