import { runTrace } from '../trace.ts';
import { stubbedWorkflow, input, finish } from './lib/load.ts';

// SubmitJob -> WaitForCompletion -> GetJobStatus -> DetermineCompletion.
// First status check answers `undefined` (neither guard matches: unguarded third
// transition back to WaitForCompletion), the second answers SUCCEEDED -> JobSucceeded -> End.
const logic = stubbedWorkflow([undefined, 'SUCCEEDED']);

const t = await runTrace(
  'workflow-monitor-job/succeeded',
  logic,
  [
    { send: { type: 'unknown' } }, // no transition in SubmitJob
    { wait: 30 }, // submitJob pending (resolves at 100 ms)
    { wait: 120 }, // submitJob done -> WaitForCompletion, jobuid assigned
    { advance: 4999 }, // the 5000 ms delay has not elapsed
    { advance: 1 }, // after 5000 -> GetJobStatus
    { wait: 30 }, // checkJobStatus pending
    { wait: 120 }, // jobStatus undefined: always falls through to WaitForCompletion
    { advance: 5000 }, // after 5000 -> GetJobStatus (second check)
    { wait: 150 }, // jobStatus SUCCEEDED -> JobSucceeded (report pending)
    { wait: 120 }, // reportJobSucceeded done -> End (final)
    { advance: 5000 } // no timer is left: stays done
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
