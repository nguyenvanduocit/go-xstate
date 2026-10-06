// Todo.tsx and Todos.tsx import React, @xstate/react and classnames, which are not installed
// here, so the example files cannot be imported. The pure logic they hold (todoMachine,
// filterTodos) is cut out of the unmodified source text and evaluated with the same
// `assign` / `setup` from xstate.
import { resolve } from 'node:path';
import { assign, setup } from 'xstate';

const srcDir = resolve(import.meta.dir, '../../../../references/xstate/examples/todomvc-react/src');
const transpiler = new Bun.Transpiler({ loader: 'ts' });

async function cut(file: string, from: string, to: string, name: string): Promise<string> {
  const text = await Bun.file(`${srcDir}/${file}`).text();
  const start = text.indexOf(from);
  const end = text.indexOf(to, start);
  if (start < 0 || end < 0) throw new Error(`cannot find ${name} in ${file}`);
  return text.slice(start, end).replace(/^export /, '');
}

function evaluate(tsSource: string, name: string): any {
  const js = transpiler.transformSync(tsSource);
  return new Function('assign', 'setup', `${js}\nreturn ${name};`)(assign, setup);
}

// Todo.tsx: `export const todoMachine = setup({...}).createMachine({...});`
export const todoMachine = evaluate(
  await cut('Todo.tsx', 'export const todoMachine', 'export function Todo(', 'todoMachine'),
  'todoMachine'
);

// Todos.tsx: `function filterTodos(filter, todos) {...}`
export const filterTodos: (filter: string, todos: any[]) => any[] = evaluate(
  await cut('Todos.tsx', 'function filterTodos', 'export function Todos(', 'filterTodos'),
  'filterTodos'
);
