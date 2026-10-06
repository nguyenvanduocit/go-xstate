import { createStore } from '@xstate/store';
import { counterConfig, steps } from './lib/counterStore.ts';

// The real code path of App.tsx: createStore(config), `store.inspect(fn)` (here
// a recorder instead of createBrowserInspector().inspect), and the selector
// `(s) => s.context.count` (what useSelector subscribes to).
// Golden: { name, steps: [ { step, snapshot: {status, context, count, inspected, notified} } ] }
//   inspected: inspection events received during the step: { type, event, context }
//   notified:  count values the selector subscription received during the step
const store = createStore(counterConfig);
const count = store.select((ctx: { count: number }) => ctx.count);

let inspected: any[] = [];
let notified: number[] = [];
store.inspect((e: any) => {
  inspected.push({ type: e.type, event: e.event, context: e.snapshot.context });
});
count.subscribe((n: number) => notified.push(n));

const view = () => {
  const s = store.getSnapshot();
  const out = JSON.parse(
    JSON.stringify({ status: s.status, context: s.context, count: count.get(), inspected, notified })
  );
  inspected = [];
  notified = [];
  return out;
};

const recorded: any[] = [{ step: 'start', snapshot: view() }];
for (const step of steps) {
  store.send(step.send as any);
  recorded.push({ step, snapshot: view() });
}
console.log(JSON.stringify({ name: 'store-counter-react/store', steps: recorded }, null, 2));
