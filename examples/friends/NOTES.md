# friends-list-react

Ported from `references/xstate/examples/friends-list-react/src/friendMachine.ts` (`friend.go`) and
`src/friendsMachine.ts` (`friends.go`).

## Port decisions
- `makeId()` (`Math.random().toString(36).substring(7)`, `friendsMachine.ts:4`) becomes the `makeID func() string`
  parameter of `FriendsMachine`. The JS traces replace `Math.random` with a fixed sequence; the Go tests pass the same
  ids (`xkxayr`, `bta69r`, ...) as inputs of `seqIDs` in `trace_test.go`.
- `event.name.trim().length > 0` (`friendsMachine.ts:38`) is `jsTrim` in `friends.go`, not `strings.TrimSpace`: JS trims
  U+FEFF but not U+0085, Go is the other way round. The `friends` trace covers both.
- `context.friends[event.index]` with an index outside the list yields `undefined` in JS; `friendAt` returns a nil
  `ActorRef`, which `StopChild` ignores like JS does.
- `saveUser` waits 1 s as in JS; the Go promise also honours cancellation (JS lets the timer run and ignores the result
  once the actor is stopped; the observable behaviour is the same).

## Not ported
- `src/App.tsx`, `src/Friend.tsx`, `src/main.tsx`: React components and `@xstate/react` hooks (`useMachine`, `useActor`).
- `src/App.css`, `src/index.css`, `src/favicon.svg`, `src/vite-env.d.ts`, `index.html`, `vite.config.ts`, `tsconfig.json`,
  `package.json`, `pnpm-lock.yaml`, `readme.md`: styling, HTML shell and bundler/TypeScript config.
- JS type declarations (`types: {...}`, `ActorRefFrom`): Go types `FriendContext`, `FriendsContext`, `FriendInput`.
- Guard behaviour when `FRIENDS.ADD`/`NEW_FRIEND.CHANGE` events lack a string `name`: JS throws a TypeError in the guard;
  the Go port treats a missing or non-string name as `""` (guard false). The trace driver only sends well-formed events.

## Trace coverage
All goldens are recorded from the JS reference by `scripts/trace/friends-list-react/*.ts`
(`scripts/trace/gen.sh friends-list-react`); input and `clock` are written into the golden by the scripts.

`testdata/friend.golden.json` (`friend.ts`, input `{name: 'Ann'}`, `saveUser` replaced by a 30 ms promise on both sides):
- states `reading`, `editing`, `saving` and their tags `read` / `form` / `form,saving`.
- `EDIT`, `SET_NAME` (handled in `editing`, ignored in `reading` and `saving`), `SAVE`, `CANCEL` from each of `reading`,
  `editing` (name reverts to `prevName`) and `saving` (invoked promise discarded, nothing happens when it settles).
- `invoke.onDone` (`saving -> reading`, `prevName = name`), an event not handled in the current state, an unknown event.

`testdata/friends.golden.json` (`friends.ts`, fixed ids): root machine only (no states, `value` is `{}`).
- `NEW_FRIEND.CHANGE`; `FRIENDS.ADD` guard false (empty, spaces, `﻿`, NBSP+tab) and true (`Alice`, `x` with empty
  `newFriendName`, `\u0085`); `FRIEND.REMOVE` with index out of range, negative, non-integer, and valid (middle, first,
  until the list is empty and once more on the empty list); a spawn after removals; unknown event.
- `context.friends` serialises as `{xstate$$type: 1, id}`; `children` lists the spawned ids.

`testdata/friends-children.golden.json` (`friends-children.ts`, replayed by `children_test.go` with its own driver
because `tracetest.Run` can only send to the root): friend actors spawned by `friendsMachine`, real `saveUser`.
- `input.name` wiring from `context.newFriendName` (`Dana`, and `''` when `FRIENDS.ADD` carries a name but the context has none).
- full friend lifecycle through the parent, `saveUser` resolving after 1 s (real-time `wait: 1100`).
- `stopChild`: a friend removed while `saving` ends `stopped` and its promise never completes; the friend removed last
  is `stopped` too; remaining friends keep working.
