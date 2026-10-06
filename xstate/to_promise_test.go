package xstate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JS: toPromise > should be awaitable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L4
func TestToPromise_ShouldBeAwaitable(t *testing.T) {
	promiseActor := xs.CreateActor(
		xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
	).Start()

	result, err := xs.ToPromise(promiseActor).Wait()
	require.NoError(t, err)

	// `result satisfies number` is a type-level check; the runtime type is asserted by Equal below.
	assert.Equal(t, 42, result)
}

// JS: toPromise > should await actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L16
func TestToPromise_ShouldAwaitActors(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{
				"RESOLVE": {{Target: "done"}},
			}},
			{Key: "done", Type: xs.Final},
		},
		Output: map[string]any{"count": 42},
	})

	actor := xs.CreateActor(machine).Start()

	time.AfterFunc(ms(1), func() {
		actor.Send(xs.Ev("RESOLVE"))
	})

	data, err := xs.ToPromise(actor).Wait()
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"count": 42}, data)
}

// JS: toPromise > should await already done actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L48
func TestToPromise_ShouldAwaitAlreadyDoneActors(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "done",
		States: xs.States{
			{Key: "done", Type: xs.Final},
		},
		Output: map[string]any{"count": 42},
	})

	actor := xs.CreateActor(machine).Start()

	data, err := xs.ToPromise(actor).Wait()
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"count": 42}, data)
}

// JS: toPromise > should handle errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L71
func TestToPromise_ShouldHandleErrors(t *testing.T) {
	t.Skip("skipped in JS")

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{
				"REJECT": {{Actions: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
					panic(errors.New("oh noes"))
				})}}},
			}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	time.AfterFunc(ms(1), func() {
		actor.Send(xs.Ev("REJECT"))
	})

	if _, err := xs.ToPromise(actor).Wait(); err != nil {
		assert.Equal(t, errors.New("oh noes"), err)
	}
}

// JS: toPromise > should immediately resolve for a done actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L100
func TestToPromise_ShouldImmediatelyResolveForADoneActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "done",
		States: xs.States{
			{Key: "done", Type: xs.Final},
		},
		Output: map[string]any{"count": 100},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusDone, actor.GetSnapshot().Status)
	assert.Equal(t, map[string]any{"count": 100}, actor.GetSnapshot().Output)

	output, err := xs.ToPromise(actor).Wait()
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"count": 100}, output)
}

// JS: toPromise > should immediately reject for an actor that had an error
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/toPromise.test.ts#L123
func TestToPromise_ShouldImmediatelyRejectForAnActorThatHadAnError(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
			panic(errors.New("oh noes"))
		})},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(_ any) {},
	})
	actor.Start()

	assert.Equal(t, xs.StatusError, actor.GetSnapshot().Status)
	assert.Equal(t, errors.New("oh noes"), actor.GetSnapshot().Error)

	_, err := xs.ToPromise(actor).Wait()
	assert.Equal(t, errors.New("oh noes"), err)
}
