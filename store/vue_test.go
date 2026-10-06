package store_test

import (
	"testing"
)

// JS: works with `useSelector(…)` (@xstate/vue)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/vue.test.ts#L6
func TestStoreVue_WorksWithUseSelectorXstateVue(t *testing.T) {
	t.Skip("N/A: Vue component rendering (@testing-library/vue render + fireEvent on UseSelector.vue); no Go equivalent")
}

// JS: works with `useActor(…)` (@xstate/vue)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/vue.test.ts#L18
func TestStoreVue_WorksWithUseActorXstateVue(t *testing.T) {
	t.Skip("N/A: Vue component rendering (@testing-library/vue render + fireEvent on UseActor.vue); no Go equivalent")
}

// JS: works with `useActorRef(…)` (@xstate/vue)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/vue.test.ts#L30
func TestStoreVue_WorksWithUseActorRefXstateVue(t *testing.T) {
	t.Skip("N/A: Vue component rendering (@testing-library/vue render + fireEvent on UseActorRef.vue); no Go equivalent")
}
