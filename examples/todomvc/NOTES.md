# todomvc-react

Ported from `references/xstate/examples/todomvc-react/src/`:
- `todosMachine.ts` -> `machine.go` (`Machine`, `Context`, `TodoItem`).
- `Todo.tsx` `todoMachine` (lines 8-75) and its `provide()` wiring (lines 79-100) -> `todo.go` (`TodoMachine`, `TodoMachineFor`).
- `Todos.tsx` `filterTodos` (8-17), the per-render derivations (53-56, 148) and the URL-hash handling (40, 45-51) -> `view.go`
  (`FilterTodos`, `Derive`, `FilterFromHash`, `InitialHashEvent`).

## Port decisions
- `Math.random().toString(36).substring(7)` (`todosMachine.ts:53`) becomes the `newID func() string` parameter of `Machine`.
  The JS traces replace `Math.random` with a fixed sequence; the Go tests pass the same ids (`xkxayr`, `bta69r`, ...) in `seqIDs`.
- `String.prototype.trim` (`todosMachine.ts:48`, `:67`) is `jsTrim` in `machine.go`, not `strings.TrimSpace`: JS trims U+FEFF
  but not U+0085, Go is the other way round. The `todos` trace covers both.
- `todo.commit` carries a `todo` object; `itemOf` accepts a `TodoItem` (the Go wiring in `TodoMachineFor`) and the decoded-JSON map (goldens).
  JS replaces the matching item with the event's object as is (extra fields included); Go keeps `id`, `title`, `completed` only.
- `filter.change` stores any string (`todosMachine.ts:93`); an unknown filter makes `filterTodos` return every todo.
- `filterTodos` returns the same array in the `all`/unknown case; `FilterTodos` returns a copy (equal content).
- `Todo.tsx` provides `onCommit` and `focusInput`; `TodoMachine` has the empty setup() versions, `TodoMachineFor(todos, todo)`
  provides `onCommit` like the component does. `todo` is the first-render item (React `useActorRef` creates the actor once), so a
  commit after `todo.mark` sends back the original `completed`.

## Not ported
- `src/App.tsx`, `src/Todos.tsx` and `src/Todo.tsx` JSX, `src/main.tsx`: React components and `@xstate/react` hooks (`createActorContext`, `useActorRef`, `useSelector`), DOM events, `classnames`, `useRef`/focus handling.
- `focusInput` side effect (`setTimeout` + `input.select()`, `Todo.tsx:91-95`): DOM only; the action stays an empty function.
- `src/useHashChange.ts`, `window.location.hash` and the `hashchange` listener: browser APIs. The hash-to-filter rule is ported (`FilterFromHash`, `InitialHashEvent`), the listener is not.
- `localStorage` access (`App.tsx:7`, `Todos.tsx:28-34`): the persisted JSON snapshot round trip is exercised (`TestPersistedTrace`), the storage itself is not.
- `src/App.css`, `src/index.css`, `src/logo.svg`, `src/react-app-env.d.ts`, `index.html`, `vite.config.ts`, `tsconfig.json`, `package.json`, `pnpm-lock.yaml`, `readme.md`, `.gitignore`: styling, HTML shell, bundler and TypeScript config.
- JS `types: {...}` declarations: Go types `Context`, `TodoItem`, `TodoContext`, `TodoInput`.
- Events with missing or mistyped payload (`value` not a string, `filter` missing, `todo` not an object): JS throws a TypeError or stores `undefined`; Go reads a missing or non-string field as `""` / a zero `TodoItem`. The traces only send well-formed events.

## Trace coverage
Goldens are recorded from the JS reference by `scripts/trace/todomvc-react/*.ts` (`scripts/trace/gen.sh todomvc-react`).
`Todo.tsx`/`Todos.tsx` import React and cannot be loaded under bun, so `lib/source.ts` cuts `todoMachine` and `filterTodos` out of the unmodified
source text and evaluates them with the repo's `xstate`. `derived.ts` copies the three component-body expressions (`Todos.tsx:53-55`, `:148`) by hand;
`todo-commit.ts` copies the `onCommit` provide of `Todo.tsx:82-90` by hand.

`testdata/todos.golden.json` (`todos.ts`, `TestTodosTrace`): the root machine has no states (`value` is `{}`).
- `newTodo.change`; `newTodo.commit` guard false (empty, spaces, U+FEFF) and true (plain, surrounding spaces kept, U+0085); fresh ids from the pinned `Math.random`.
- `todo.commit` replace by id, unknown id, blank title removes (spaces, U+FEFF).
- `todo.mark` completed/active, unknown id; `todo.markAll` completed/active, also on the empty list.
- `todo.delete` existing, unknown id, down to the empty list; `todos.clearCompleted` with and without completed items, also on the empty list.
- `filter.change` all/active/completed and an unvalidated value; unknown event.

`testdata/todo.golden.json` (`todo.ts`, `TestTodoTrace`, input `{todo}`): `reading` and `editing`; `edit` (also ignored in `editing`), `change`
(ignored in `reading`, where it is not declared), `cancel` (reverts to `initialTitle`; not handled in `reading`), `blur` (commit; not handled in `reading`),
entry assign of `initialTitle`, unknown event. `onCommit` is a no-op here.

`testdata/todo-commit.golden.json` (`todo-commit.ts`, `TestTodoCommitTrace`): one todo actor wired to the todos actor with the real `onCommit`:
commit changes the title, cancel does not commit, a commit after `todo.mark` resets `completed` to the first-render value, a blank title removes
the todo, and a commit for a todo that no longer exists changes nothing.

`testdata/persisted.golden.json` (`persisted.ts`, `TestPersistedTrace`): `getPersistedSnapshot()` after every event of session 1, then session 2
restores the JS-written snapshot (`createActor(machine, {snapshot})`) and continues. Null `output`/`error` are dropped before comparing, see
`docs/porting/notes/examples-lib-findings.md` (persisted-donut-maker).

`testdata/derived.golden.json` (`derived.ts`, `TestDerived`): `filterTodos` x 4 filters x 6 todo lists (empty, one active, one completed, mixed,
all completed, the initial context), `numActiveTodos`, `allCompleted`, `mark`, clear-button visibility, and 9 URL hashes (`''`, `#`, `#/`, `#/active`,
`#/completed`, `#/all`, `#/bogus`, `#x`, `#/a/b`) through `FilterFromHash` and `InitialHashEvent`.
