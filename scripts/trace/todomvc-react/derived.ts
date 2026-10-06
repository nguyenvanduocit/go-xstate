import { todosMachine } from '../../../references/xstate/examples/todomvc-react/src/todosMachine.ts';
import { filterTodos } from './lib/source.ts';

// What Todos.tsx derives from the actor context on every render. `filterTodos` is the real
// function from Todos.tsx; the three expressions below are Todos.tsx:53-55 and :148 copied verbatim
// (they live inside the component body and cannot be imported).
const derive = (filter: string, todos: any[]) => {
  const numActiveTodos = todos.filter((todo) => !todo.completed).length;
  const allCompleted = todos.length > 0 && numActiveTodos === 0;
  const mark = !allCompleted ? 'completed' : 'active';
  return {
    filtered: filterTodos(filter, todos),
    numActiveTodos,
    allCompleted,
    mark,
    clearCompletedVisible: numActiveTodos < todos.length
  };
};

// Hash handling of Todos.tsx:40 and :46-49 (`window.location.hash.slice(2) || 'all'`).
const filterFromHash = (hash: string) => (hash.slice(2) || 'all');
const initialHashEvent = (hash: string) =>
  hash.slice(2) ? { type: 'filter.change', filter: hash.slice(2) } : null;

const todo = (id: string, completed: boolean) => ({ id, title: `todo ${id}`, completed });
const lists: any[][] = [
  [],
  [todo('a', false)],
  [todo('a', true)],
  [todo('a', false), todo('b', true), todo('c', false)],
  [todo('a', true), todo('b', true)],
  todosMachine.config.context.todos
];
const cases: any[] = [];
for (const todos of lists)
  for (const filter of ['all', 'active', 'completed', 'bogus'])
    cases.push({ filter, todos, derived: derive(filter, todos) });

const hashes = ['', '#', '#/', '#/active', '#/completed', '#/all', '#/bogus', '#x', '#/a/b'];
const hashCases = hashes.map((hash) => ({
  hash,
  filter: filterFromHash(hash),
  initialEvent: initialHashEvent(hash)
}));

console.log(JSON.stringify({ name: 'todomvc-react/derived', cases, hashCases }, null, 2));
