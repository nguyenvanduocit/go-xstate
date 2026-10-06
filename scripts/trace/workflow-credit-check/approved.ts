import { runTrace } from '../trace.ts';
import { workflow, input, creditCheckStub, startApplicationStub, rejectionEmailStub, finish } from './lib/load.ts';

// Approved: CheckCredit -> (always) EvaluateDecision -> StartApplication -> End.
const logic = workflow.provide({
  actors: {
    callCreditCheckMicroservice: creditCheckStub('Approved', 700, 'Good credit score'),
    startApplicationWorkflowId: startApplicationStub,
    sendRejectionEmailFunction: rejectionEmailStub
  }
});

const t = await runTrace(
  'workflow-credit-check/approved',
  logic,
  [
    { send: { type: 'unknown' } }, // no transition in CheckCredit
    { wait: 30 }, // credit check pending (resolves at 100 ms)
    { wait: 120 }, // credit check done -> StartApplication, creditCheck assigned (resolves at 200 ms)
    { wait: 120 }, // StartApplication done -> End (final)
    { advance: 15 * 60 * 1000 } // the PT15M timer was cancelled when CheckCredit exited: stays done
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
