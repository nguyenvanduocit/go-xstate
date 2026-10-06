import { fromStore } from '@xstate/store';
import { runTrace } from '../trace.ts';
import { counterConfig, steps } from './lib/counterConfig.ts';

// <Counter initialCount={0} /> driven as an actor (fromStore uses the same
// `on` handlers as the createStore call inside useStore).
const t = await runTrace('local-store-counter-react/counter-0', fromStore(counterConfig(0)), steps);
console.log(JSON.stringify(t, null, 2));
