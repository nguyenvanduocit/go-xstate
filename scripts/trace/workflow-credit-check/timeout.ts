import { runTrace } from '../trace.ts';
import { workflow, input, hangingCreditCheckStub, startApplicationStub, rejectionEmailStub, finish } from './lib/load.ts';

// The credit check never answers: after PT15M (15 * 60 * 1000 ms) the machine goes to Timeout
// and the pending invoke is stopped.
const logic = workflow.provide({
  actors: {
    callCreditCheckMicroservice: hangingCreditCheckStub,
    startApplicationWorkflowId: startApplicationStub,
    sendRejectionEmailFunction: rejectionEmailStub
  }
});

const t = await runTrace(
  'workflow-credit-check/timeout',
  logic,
  [
    { advance: 15 * 60 * 1000 - 1 }, // one ms short of the timeout: still CheckCredit
    { advance: 1 }, // PT15M fires -> Timeout (no children left)
    { send: { type: 'unknown' } } // Timeout has no transitions
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
