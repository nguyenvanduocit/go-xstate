import { createStore } from '@xstate/store';
import { counterConfig, steps } from './lib/counterConfig.ts';

// The real code path of App.tsx: createStore(config) (what useStore calls) plus
// the selector `(s) => s.context.count` (what useSelector subscribes to).
// Golden: { name, initialCount, steps: [ { step, snapshot: {status, context, count} } ] }
const runs = [0, 10, 100].map((initialCount) => {
  const store = createStore(counterConfig(initialCount));
  const count = store.select((ctx: { count: number }) => ctx.count);
  const view = () => {
    const s = store.getSnapshot();
    return JSON.parse(
      JSON.stringify({ status: s.status, context: s.context, count: count.get() })
    );
  };
  const recorded: any[] = [{ step: 'start', snapshot: view() }];
  for (const step of steps) {
    store.send(step.send as any);
    recorded.push({ step, snapshot: view() });
  }
  return { initialCount, steps: recorded };
});
console.log(JSON.stringify({ name: 'local-store-counter-react/store', runs }, null, 2));
