package friends_test

import (
	"context"
	"testing"
	"time"

	friends "github.com/nguyenvanduocit/go-xstate/examples/friends"
	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// seqIDs returns a makeID stub yielding ids in order. The values are the inputs
// the JS traces feed through a stubbed Math.random:
// (0.1234567890123).toString(36).substring(7) === "xkxayr", and so on.
func seqIDs(ids ...string) func() string {
	i := 0
	return func() string {
		id := ids[i]
		i++
		return id
	}
}

// TestFriendTrace replays testdata/friend.golden.json, recorded by
// scripts/trace/friends-list-react/friend.ts. Both sides replace saveUser
// with a promise that resolves true after 30 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/friends-list-react/src/friendMachine.ts#L3
// JS trace: scripts/trace/friends-list-react/friend.ts.
func TestFriendTrace(t *testing.T) {
	machine := friends.FriendMachine().Provide(xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"saveUser": xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (bool, error) {
				select {
				case <-time.After(30 * time.Millisecond):
					return true, nil
				case <-ctx.Done():
					return false, context.Cause(ctx)
				}
			}),
		},
	})
	tracetest.Run(t, "testdata/friend.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[friends.FriendContext]] {
		return xs.CreateActor(machine, xs.WithInput(input))
	})
}

// TestFriendsTrace replays testdata/friends.golden.json, recorded by
// scripts/trace/friends-list-react/friends.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/friends-list-react/src/friendsMachine.ts#L6
// JS trace: scripts/trace/friends-list-react/friends.ts.
func TestFriendsTrace(t *testing.T) {
	machine := friends.FriendsMachine(seqIDs("xkxayr", "bta69r", "a4v5h4", "tcp7oc", "5j23v", "janbwe"))
	tracetest.Run(t, "testdata/friends.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[friends.FriendsContext]] {
		return xs.CreateActor(machine)
	})
}
