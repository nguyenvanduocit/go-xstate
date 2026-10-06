import { record } from './lib/record.ts';

await record('rejected', [
  { send: { type: 'visaRejectedEvent' } }, // -> HandleRejectedVisa
  { advance: 5000 }, // the cancelled after timer must not fire
  { wait: 50 }, // promise still pending
  { send: { type: 'visaApprovedEvent' } }, // ignored in HandleRejectedVisa
  { wait: 150 } // promise resolved: onDone -> End (final)
]);
