import { record } from './lib/record.ts';

await record('timeout', [
  { advance: 500 }, // still CheckVisaStatus
  { advance: 499 }, // 999 ms: still CheckVisaStatus
  { advance: 1 }, // 1000 ms: after visaDecisionTimeout -> HandleNoVisaDecision
  { send: { type: 'visaApprovedEvent' } }, // ignored in HandleNoVisaDecision
  { wait: 50 }, // promise still pending
  { wait: 150 } // promise resolved: onDone -> End (final)
]);
