import { todoMachine } from './lib/source.ts';
import { runTrace } from '../trace.ts';

// todoMachine on its own with the two actions Todo.tsx provides stubbed as no-ops
// (onCommit and focusInput have no observable effect on the snapshot here;
// todo-commit.ts records the real onCommit wiring).
const machine = todoMachine.provide({ actions: { onCommit: () => {}, focusInput: () => {} } });
const input = { todo: { id: '1', title: 'Learn state machines', completed: false } };
const change = (value: string) => ({ send: { type: 'change', value } });
const t = await runTrace(
  'todomvc-react/todo',
  machine,
  [
    change('ignored in reading'),
    { send: { type: 'cancel' } }, // not handled in reading
    { send: { type: 'blur' } }, // not handled in reading
    { send: { type: 'edit' } }, // reading -> editing: initialTitle = title
    { send: { type: 'edit' } }, // not handled in editing
    change('Learn XState'),
    change('Learn XState v5'),
    { send: { type: 'cancel' } }, // editing -> reading, title = initialTitle
    { send: { type: 'edit' } },
    change('Kept'),
    { send: { type: 'blur' } }, // editing -> reading with onCommit
    { send: { type: 'edit' } }, // initialTitle = 'Kept'
    change(''),
    { send: { type: 'cancel' } },
    { send: { type: 'unknown' } }
  ],
  { input }
);
console.log(JSON.stringify({ name: t.name, clock: false, input, steps: t.steps }, null, 2));
