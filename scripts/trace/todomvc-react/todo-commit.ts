import { createActor } from 'xstate';
import { todosMachine } from '../../../references/xstate/examples/todomvc-react/src/todosMachine.ts';
import { todoMachine } from './lib/source.ts';
import { view } from '../trace.ts';

// One <Todo todo={...}/> wired to the todos actor exactly like Todo.tsx does:
// useActorRef(todoMachine.provide({ actions: { onCommit, focusInput } }), { input: { todo } })
// where onCommit sends `todo.commit` with `{ ...todo, title: context.title }`.
// `todo` is the prop of the first render: useActorRef creates the actor once, so the
// closure keeps that value even when the todos context changes later.
const todosActor = createActor(todosMachine).start();
const todo = todosActor.getSnapshot().context.todos[0];
const todoActor = createActor(
  todoMachine.provide({
    actions: {
      onCommit: ({ context }: any) => {
        todosActor.send({ type: 'todo.commit', todo: { ...todo, title: context.title } });
      },
      focusInput: () => {}
    }
  }),
  { input: { todo } }
).start();

const snapshot = () => ({ todo: view(todoActor.getSnapshot()), todos: view(todosActor.getSnapshot()) });
const out: any[] = [{ step: 'start', snapshot: snapshot() }];
const step = (to: 'todo' | 'todos', event: Record<string, unknown>) => {
  (to === 'todo' ? todoActor : todosActor).send(event as any);
  out.push({ step: { to, send: event }, snapshot: snapshot() });
};

step('todo', { type: 'edit' });
step('todo', { type: 'change', value: 'Learn XState' });
step('todo', { type: 'blur' }); // commit: title changes in todos
step('todo', { type: 'edit' });
step('todo', { type: 'change', value: 'discarded' });
step('todo', { type: 'cancel' }); // no commit
step('todos', { type: 'todo.mark', id: '1', mark: 'completed' });
step('todo', { type: 'edit' });
step('todo', { type: 'change', value: 'Learn XState again' });
step('todo', { type: 'blur' }); // commit resends the first-render todo: completed goes back to false
step('todo', { type: 'edit' });
step('todo', { type: 'change', value: '   ' });
step('todo', { type: 'blur' }); // blank title: todo.commit removes the todo
step('todo', { type: 'edit' });
step('todo', { type: 'change', value: 'Back again' });
step('todo', { type: 'blur' }); // the todo no longer exists in todos: nothing to replace
todoActor.stop();
todosActor.stop();
console.log(JSON.stringify({ name: 'todomvc-react/todo-commit', steps: out }, null, 2));
