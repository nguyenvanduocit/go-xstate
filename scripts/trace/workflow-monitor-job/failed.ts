import { runTrace } from '../trace.ts';
import { stubbedWorkflow, input, finish } from './lib/load.ts';

// First status check answers FAILED: second guard -> JobFailed -> End.
const logic = stubbedWorkflow(['FAILED']);

const t = await runTrace(
  'workflow-monitor-job/failed',
  logic,
  [
    { wait: 120 }, // submitJob done -> WaitForCompletion
    { advance: 5000 }, // after 5000 -> GetJobStatus
    { wait: 120 }, // jobStatus FAILED -> JobFailed (report pending)
    { wait: 120 } // reportJobFailed done -> End (final)
  ],
  { input, useClock: true }
);
finish({ ...t, clock: true, input });
