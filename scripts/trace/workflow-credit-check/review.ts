import { runTrace } from '../trace.ts';
import { workflow, input, creditCheckStub, startApplicationStub, rejectionEmailStub, finish } from './lib/load.ts';

// A decision that is neither Approved nor Denied: both guards fail, the unguarded third
// transition of EvaluateDecision goes to RejectApplication.
const logic = workflow.provide({
  actors: {
    callCreditCheckMicroservice: creditCheckStub('Review', 600, 'Needs manual review'),
    startApplicationWorkflowId: startApplicationStub,
    sendRejectionEmailFunction: rejectionEmailStub
  }
});

const t = await runTrace(
  'workflow-credit-check/review',
  logic,
  [
    { wait: 30 }, // credit check pending
    { wait: 120 }, // credit check done -> RejectApplication via the fallback transition
    { wait: 120 } // rejection email sent -> End (final)
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
