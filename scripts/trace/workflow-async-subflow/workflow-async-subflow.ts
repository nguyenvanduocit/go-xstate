import { runTrace } from '../trace.ts';
import { workflow, onboarding, promptStub, answersFor, finish } from './lib/load.ts';

// Whole workflow: `Onboard` invokes the `onboarding` machine (whose prompt actor is stubbed).
const logic = workflow.provide({
  actors: {
    onboarding: onboarding.provide({ actors: { prompt: promptStub(answersFor('Ada')) } })
  }
});

const t = await runTrace('workflow-async-subflow', logic, [
  { send: { type: 'unknown' } }, // no transition in Onboard
  { wait: 20 }, // child waits in Welcome (first prompt due at 50 ms)
  { wait: 50 }, // child in Personalize (second prompt due at 100 ms)
  { wait: 60 }, // child done: onDone -> Onboarded (final)
  { send: { type: 'unknown' } } // ignored once done
]);
finish({ ...t, clock: false });
