> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store test fixes

One line per edit of a translated store test: `file:line · JS file:line · what was wrong`.

- store/atom_test.go:782 · references/xstate/packages/xstate-store/test/atom.test.ts:693 · `count.Set(2)` then `atom.Get() == pending`: JS cannot settle the async getter between two synchronous statements (settlement is a microtask), but the Go getter runs on its own goroutine and settled in between (5/40 runs under -race); the second getter run now waits on a `gate` released after the synchronous assertion (same pattern as store_1 async-effect tests). Assertions unchanged.
- store/atom_test.go:920 · references/xstate/packages/xstate-store/test/atom.test.ts:800 · two back-to-back `Get() == pending` reads raced with the getter goroutine settling (7/40 runs under -race); getter waits on a `gate` released after both reads. Assertions unchanged.
- store/atom_test.go:956 · references/xstate/packages/xstate-store/test/atom.test.ts:837 · same race as the previous line for the rejecting getter (8/40 runs under -race); same `gate` fix. Assertions unchanged.
