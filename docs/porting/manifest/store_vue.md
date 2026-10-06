> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_vue_1 manifest

Source: `references/xstate/packages/xstate-store/test/vue.test.ts` lines 1-41.
Go file: `store/vue_test.go` (tag `port_store_vue_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| works with `useSelector(…)` (@xstate/vue) | TestStoreVue_WorksWithUseSelectorXstateVue | N/A-runtime | Renders UseSelector.vue with @testing-library/vue and clicks a DOM button; Vue binding only |
| works with `useActor(…)` (@xstate/vue) | TestStoreVue_WorksWithUseActorXstateVue | N/A-runtime | Renders UseActor.vue; Vue binding only |
| works with `useActorRef(…)` (@xstate/vue) | TestStoreVue_WorksWithUseActorRefXstateVue | N/A-runtime | Renders UseActorRef.vue; Vue binding only |

## API gaps

None (no `apigap_store_vue_1.go`).

## Notes for reviewers

All three tests assert only DOM text content (`textContent` '0' then '1') after a Vue component click; they exercise `@xstate/vue` hooks, not store logic. The `.vue` fixtures are not part of this chunk.
