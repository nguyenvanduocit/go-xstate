import { record } from './lib/record.ts';

await record('approved', [
  { send: { type: 'unknown' } }, // ignored in CheckVisaStatus
  { advance: 999 }, // 1 ms before visaDecisionTimeout: still CheckVisaStatus
  { send: { type: 'visaApprovedEvent' } }, // -> HandleApprovedVisa (after timer cancelled)
  { advance: 5000 }, // the cancelled after timer must not fire
  { wait: 50 }, // promise still pending
  { send: { type: 'visaRejectedEvent' } }, // ignored in HandleApprovedVisa
  { wait: 150 }, // promise resolved: onDone -> End (final)
  { send: { type: 'visaApprovedEvent' } } // ignored once done
]);
