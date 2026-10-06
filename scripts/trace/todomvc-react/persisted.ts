import { createActor } from 'xstate';
import { todosMachine } from '../../../references/xstate/examples/todomvc-react/src/todosMachine.ts';
import { view } from '../trace.ts';

// App.tsx persists with
//   todosActorRef.subscribe(() => localStorage.setItem('todos', JSON.stringify(todosActorRef.getPersistedSnapshot())))
// and restores with createActorContext(todosMachine, { state: JSON.parse(localStorage.getItem('todos') || 'null') }).
// Session 1 records the persisted JSON after every event; session 2 restores the final one and continues.
const randoms = [0.1234567890123, 0.2345678901234];
let next = 0;
Math.random = () => randoms[next++];

type Ev = Record<string, unknown>;
const session1: Ev[] = [
  { type: 'newTodo.change', value: 'Second' },
  { type: 'newTodo.commit', value: 'Second' },
  { type: 'todo.mark', id: '1', mark: 'completed' },
  { type: 'filter.change', filter: 'completed' }
];
const session2: Ev[] = [
  { type: 'todos.clearCompleted' },
  { type: 'newTodo.commit', value: 'Third' },
  { type: 'filter.change', filter: 'all' }
];

const run = (events: Ev[], persisted: any) => {
  const actor = createActor(todosMachine, persisted ? { snapshot: persisted } : {});
  let last: any = null;
  // localStorage.setItem(...) runs on every snapshot emission
  actor.subscribe(() => {
    last = JSON.parse(JSON.stringify(actor.getPersistedSnapshot()));
  });
  actor.start();
  const steps: any[] = [{ step: 'start', snapshot: view(actor.getSnapshot()), persisted: last }];
  for (const event of events) {
    actor.send(event as any);
    steps.push({ step: { send: event }, snapshot: view(actor.getSnapshot()), persisted: last });
  }
  actor.stop();
  return steps;
};

const first = run(session1, null);
const restoredFrom = first[first.length - 1].persisted;
const second = run(session2, restoredFrom);
console.log(
  JSON.stringify({ name: 'todomvc-react/persisted', sessions: [{ restoredFrom: null, steps: first }, { restoredFrom, steps: second }] }, null, 2)
);
