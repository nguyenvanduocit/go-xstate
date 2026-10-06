import { fromStore } from '@xstate/store';
import { runTrace } from '../trace.ts';
import { counterConfig, steps } from './lib/counterStore.ts';

// The App.tsx store config driven as an actor (fromStore uses the same `on`
// handlers as the createStore call in App.tsx).
const t = await runTrace('store-counter-react/counter', fromStore(counterConfig), steps);
console.log(JSON.stringify(t, null, 2));
