import { runTrace } from '../trace.ts';
import { workflow, input, creditCheckStub, startApplicationStub, rejectionEmailStub, finish } from './lib/load.ts';

// Denied: CheckCredit -> EvaluateDecision -> RejectApplication -> End (second guard branch).
const logic = workflow.provide({
  actors: {
    callCreditCheckMicroservice: creditCheckStub('Denied', 400, 'Low credit score'),
    startApplicationWorkflowId: startApplicationStub,
    sendRejectionEmailFunction: rejectionEmailStub
  }
});

const t = await runTrace(
  'workflow-credit-check/denied',
  logic,
  [
    { wait: 30 }, // credit check pending
    { wait: 120 }, // credit check done -> RejectApplication (resolves at 200 ms)
    { wait: 120 } // rejection email sent -> End (final)
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
