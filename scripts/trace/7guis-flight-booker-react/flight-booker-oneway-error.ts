import { fromPromise } from 'xstate';
import { runTrace } from '../trace.ts';
import { loadFlightBookerMachine } from './harness.mjs';

// Booker stub: rejects after 100 ms of real time, so booking.onError returns to scheduling.
// The delay keeps the machine in `booking` while the following steps run (same as the Go stub).
const machine = (await loadFlightBookerMachine()).provide({
  actors: {
    Booker: fromPromise(
      () =>
        new Promise<void>((_, reject) =>
          setTimeout(() => reject(new Error('booking failed')), 100)
        )
    )
  }
});

const t = await runTrace('7guis-flight-booker-react/flight-booker-oneway-error', machine, [
  // oneWay: BOOK_DEPART guard true (depart = TODAY) -> booking
  { send: { type: 'BOOK_DEPART' } },
  // events inside booking are ignored (including CHANGE_DEPART_DATE, handled only in scheduling)
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-20' } },
  { send: { type: 'CHANGE_TRIP_TYPE' } },
  // promise rejects -> onError -> scheduling.oneWay
  { wait: 300 },
  // booking again from scheduling works
  { send: { type: 'BOOK_DEPART' } },
  { wait: 300 }
]);
console.log(JSON.stringify(t, null, 2));
