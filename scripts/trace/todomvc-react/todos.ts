import { todosMachine } from '../../../references/xstate/examples/todomvc-react/src/todosMachine.ts';
import { runTrace } from '../trace.ts';

// newTodo.commit builds ids with Math.random().toString(36).substring(7); feed it fixed values.
// (0.1234567890123).toString(36).substring(7) === 'xkxayr', etc.
const randoms = [
  0.1234567890123, 0.2345678901234, 0.3456789012345, 0.4567890123456, 0.5678901234567,
  0.6789012345678
];
let next = 0;
Math.random = () => randoms[next++];

const change = (value: string) => ({ send: { type: 'newTodo.change', value } });
const commit = (value: string) => ({ send: { type: 'newTodo.commit', value } });
const edit = (id: string, title: string, completed = false) => ({
  send: { type: 'todo.commit', todo: { id, title, completed } }
});
const del = (id: string) => ({ send: { type: 'todo.delete', id } });
const filter = (f: string) => ({ send: { type: 'filter.change', filter: f } });
const mark = (id: string, m: string) => ({ send: { type: 'todo.mark', id, mark: m } });
const markAll = (m: string) => ({ send: { type: 'todo.markAll', mark: m } });

const t = await runTrace('todomvc-react/todos', todosMachine, [
  change('Buy milk'),
  change(''),
  commit(''), // guard false: empty
  commit('   '), // guard false: whitespace only
  commit('﻿'), // guard false: U+FEFF is JS whitespace
  change('typing'),
  commit('Buy milk'), // guard true: id xkxayr, draft cleared
  commit(' Walk the dog '), // guard true: title keeps its spaces, id bta69r
  commit('\u0085'), // guard true: U+0085 is not JS whitespace, id a4v5h4
  edit('xkxayr', 'Buy oat milk'), // todo.commit: replace by id
  edit('nope', 'Ghost'), // todo.commit: unknown id, nothing changes
  edit('bta69r', '   '), // todo.commit: blank title removes the todo
  edit('a4v5h4', '﻿'), // todo.commit: U+FEFF title removes the todo
  mark('1', 'completed'),
  mark('xkxayr', 'completed'),
  mark('nope', 'completed'), // unknown id
  mark('1', 'active'),
  filter('active'),
  filter('completed'),
  filter('bogus'), // not validated by the machine
  filter('all'),
  markAll('completed'),
  markAll('active'),
  mark('xkxayr', 'completed'),
  { send: { type: 'todos.clearCompleted' } }, // removes xkxayr
  { send: { type: 'todos.clearCompleted' } }, // nothing completed: no change
  commit('Fourth'), // id tcp7oc
  del('1'),
  del('nope'), // unknown id
  del('tcp7oc'),
  markAll('completed'), // empty list
  { send: { type: 'todos.clearCompleted' } }, // empty list
  commit('Fifth'), // id 5j23v, after the list was emptied
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify({ name: t.name, clock: false, input: null, steps: t.steps }, null, 2));
