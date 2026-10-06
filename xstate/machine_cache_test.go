package xstate_test

import (
	"runtime"
	"testing"
	"time"
	"weak"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Go lifetime regression: JS uses weakly held memoization; a stopped actor must not root its machine.
// No JS GC test counterpart; JS weak-cache implementation: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/memo.ts#L1
func TestMachineCandidateCacheDoesNotRetainMachines(t *testing.T) {
	makeRefs := func() []weak.Pointer[xs.StateMachine[struct{}]] {
		base := xs.CreateMachine(xs.MachineConfig[struct{}]{})
		provided := base.Provide(xs.Implementations{})
		refs := make([]weak.Pointer[xs.StateMachine[struct{}]], 0, 2)
		for _, m := range []*xs.StateMachine[struct{}]{base, provided} {
			a := xs.CreateActor(m).Start()
			a.Send(xs.Ev("lookup"))
			a.Stop()
			refs = append(refs, weak.Make(m))
		}
		return refs
	}
	refs := makeRefs()
	require.Eventually(t, func() bool {
		runtime.GC()
		for _, ref := range refs {
			if ref.Value() != nil {
				return false
			}
		}
		return true
	}, 2*time.Second, 10*time.Millisecond, "unused machines remain strongly reachable after candidate lookup")
}
