package xstate_test

import (
	"sync"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Go concurrency regression; JS runs relay on one thread.
// Related JS ActorRef send test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3104
func TestSendToCrossRootConcurrentDelivery(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		name := "immediate"
		if delayed {
			name = "delayed"
		}
		t.Run(name, func(t *testing.T) {
			receiver := xs.CreateActor(xs.FromTransition(func(n int, _ xs.Event, _ *xs.ActorScope) int { return n + 1 }, nil)).Start()
			defer receiver.Stop()
			clock := xs.NewSimulatedClock()
			action := xs.SendTo(receiver, xs.Ev("count"))
			if delayed {
				action = xs.SendTo(receiver, xs.Ev("count"), xs.SendOptions{Delay: time.Millisecond})
			}
			sender := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[struct{}]{On: map[string]xs.Transitions{"ping": {{Actions: xs.Actions{action}}}}}), xs.WithClock(clock)).Start()
			defer sender.Stop()
			const count = 200
			var wg sync.WaitGroup
			wg.Add(3)
			go func() {
				defer wg.Done()
				for range count {
					sender.Send(xs.Ev("ping"))
					if delayed {
						clock.Increment(time.Millisecond)
					}
				}
			}()
			go func() {
				defer wg.Done()
				for range count {
					receiver.Send(xs.Ev("count"))
				}
			}()
			go func() {
				defer wg.Done()
				for range count {
					_ = receiver.GetSnapshot()
				}
			}()
			wg.Wait()
			require.Equal(t, 2*count, receiver.GetSnapshot().Context)
		})
	}
}

// Go concurrency regression: two roots must not hold each other's delivery locks.
// Related JS ActorRef send test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3104
func TestSendToCrossRootReciprocal(t *testing.T) {
	var a, b *xs.Actor[*xs.MachineSnapshot[int]]
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	machine := func(target func() xs.ActorRef) *xs.StateMachine[int] {
		return xs.CreateMachine(xs.MachineConfig[int]{On: map[string]xs.Transitions{
			"send":  {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[int]) { entered <- struct{}{}; <-release }), xs.SendTo(xs.NewExpr(func(xs.ExprArgs[int]) any { return target() }), xs.Ev("count"))}}},
			"count": {{Actions: xs.Actions{xs.Assign(func(args xs.AssignArgs[int]) int { return args.Context + 1 })}}},
		}})
	}
	a = xs.CreateActor(machine(func() xs.ActorRef { return b })).Start()
	b = xs.CreateActor(machine(func() xs.ActorRef { return a })).Start()
	done := make(chan struct{}, 2)
	go func() { a.Send(xs.Ev("send")); done <- struct{}{} }()
	go func() { b.Send(xs.Ev("send")); done <- struct{}{} }()
	for range 2 {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("source action did not start")
		}
	}
	close(release)
	for range 2 {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("reciprocal SendTo deadlocked")
		}
	}
	defer a.Stop()
	defer b.Stop()
	require.Equal(t, 1, a.GetSnapshot().Context)
	require.Equal(t, 1, b.GetSnapshot().Context)
}

// Go regression: deferred foreign delivery and its reply finish before the outer Send returns.
// Related JS ActorRef send test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3104
func TestSendToCrossRootReplyBeforeReturn(t *testing.T) {
	var sender *xs.Actor[*xs.MachineSnapshot[int]]
	receiver := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[struct{}]{On: map[string]xs.Transitions{"ping": {{Actions: xs.Actions{
		xs.SendTo(xs.NewExpr(func(xs.ExprArgs[struct{}]) any { return sender }), xs.Ev("reply")),
	}}}}})).Start()
	defer receiver.Stop()
	sender = xs.CreateActor(xs.CreateMachine(xs.MachineConfig[int]{On: map[string]xs.Transitions{
		"send":  {{Actions: xs.Actions{xs.SendTo(receiver, xs.Ev("ping"))}}},
		"reply": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[int]) int { return a.Context + 1 })}}},
	}})).Start()
	defer sender.Stop()
	sender.Send(xs.Ev("send"))
	require.Equal(t, 1, sender.GetSnapshot().Context)
}

// Go ordering regression: nested roots defer foreign sends until every source lock is released.
// Related JS ActorRef send test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3104
func TestSendToCrossRootOutermostUnlock(t *testing.T) {
	var order []string
	receiver := xs.CreateActor(xs.FromTransition(func(n int, _ xs.Event, _ *xs.ActorScope) int {
		order = append(order, "delivery")
		return n + 1
	}, nil)).Start()
	defer receiver.Stop()
	var inspected []xs.InspectionEvent
	middle := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[struct{}]{On: map[string]xs.Transitions{
		"send": {{Actions: xs.Actions{
			xs.SendTo(receiver, xs.Ev("count")),
			xs.ActionFunc(func(xs.ActionArgs[struct{}]) {
				require.Equal(t, 0, receiver.GetSnapshot().Context)
				order = append(order, "middle action")
			}),
		}}},
	}}), xs.WithInspect(func(e xs.InspectionEvent) {
		if e.Type == xs.InspectEvent && e.Event.EventType() == "count" {
			inspected = append(inspected, e)
		}
	})).Start()
	defer middle.Stop()
	outer := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[struct{}]{On: map[string]xs.Transitions{
		"send": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[struct{}]) {
			middle.Send(xs.Ev("send"))
			require.Equal(t, 0, receiver.GetSnapshot().Context, "nested Send must not flush the foreign delivery")
			order = append(order, "outer action")
		})}}},
	}})).Start()
	defer outer.Stop()
	outer.Subscribe(xs.Observer[*xs.MachineSnapshot[struct{}]]{Next: func(*xs.MachineSnapshot[struct{}]) { order = append(order, "source snapshot") }})
	outer.Send(xs.Ev("send"))
	require.Equal(t, []string{"middle action", "outer action", "source snapshot", "delivery"}, order)
	require.Equal(t, 1, receiver.GetSnapshot().Context)
	require.Len(t, inspected, 1)
	require.Same(t, middle, inspected[0].SourceRef)
	require.Same(t, receiver, inspected[0].ActorRef)
}
