// A rejection whose message matches none of the three onError guards: the error
// event is unhandled, so the machine itself errors out (status 'error').
import { workflow, record } from './lib/record.ts';
import { fromPromise } from 'xstate';

// xstate reports the unhandled error asynchronously (reportUnhandledError); keep the process alive.
process.on('uncaughtException', () => {});

const logic = workflow.provide({
  actors: {
    provisionOrderFunction: fromPromise(async () => {
      await new Promise((r) => setTimeout(r, 50));
      throw new Error('Order service unavailable');
    })
  }
});

await record('workflow-provision-orders-other-error', logic, {
  order: { id: 'o-1', item: 'laptop', quantity: '10' }
});
