import { fromPromise } from 'xstate';
import { runTrace } from '../trace.ts';
import { loadFlightBookerMachine } from './harness.mjs';

// Booker stub: resolves after 100 ms of real time (the real one sleeps 2000 ms). The delay keeps the
// machine in `booking` while the following steps run, matching the Go stub in machine_test.go.
const machine = (await loadFlightBookerMachine()).provide({
  actors: { Booker: fromPromise(() => new Promise<void>((resolve) => setTimeout(resolve, 100))) }
});

// TODAY = 2024-03-10, TOMORROW = 2024-03-11 (pinned by harness.mjs).
const t = await runTrace('7guis-flight-booker-react/flight-booker', machine, [
  // events with no handler in oneWay
  { send: { type: 'CHANGE_RETURN_DATE', value: '2024-03-20' } },
  { send: { type: 'BOOK_RETURN' } },
  // oneWay: BOOK_DEPART guard false (depart date in the past), then true
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-09' } },
  { send: { type: 'BOOK_DEPART' } },
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-12' } },
  // oneWay -> roundTrip
  { send: { type: 'CHANGE_TRIP_TYPE' } },
  { send: { type: 'BOOK_DEPART' } },
  // roundTrip: BOOK_RETURN guard false (return 03-11 <= depart 03-12, then equal)
  { send: { type: 'BOOK_RETURN' } },
  { send: { type: 'CHANGE_RETURN_DATE', value: '2024-03-12' } },
  { send: { type: 'BOOK_RETURN' } },
  // roundTrip: BOOK_RETURN guard false (return after depart but depart before today)
  { send: { type: 'CHANGE_RETURN_DATE', value: '2024-03-15' } },
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-01' } },
  { send: { type: 'BOOK_RETURN' } },
  // roundTrip -> oneWay -> roundTrip
  { send: { type: 'CHANGE_TRIP_TYPE' } },
  { send: { type: 'CHANGE_TRIP_TYPE' } },
  // BOOK_RETURN guard true -> booking -> booked (final)
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-12' } },
  { send: { type: 'BOOK_RETURN' } },
  { send: { type: 'CHANGE_DEPART_DATE', value: '2024-03-13' } },
  { wait: 300 },
  // final state ignores events
  { send: { type: 'CHANGE_TRIP_TYPE' } }
]);
console.log(JSON.stringify(t, null, 2));
