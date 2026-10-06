package xstate_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// activities1Log is a goroutine-safe ordered string log (JS `actual: string[]`).
type activities1Log struct {
	mu    sync.Mutex
	items []string
}

func (l *activities1Log) push(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, s)
}

func (l *activities1Log) get() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.items))
	copy(out, l.items)
	return out
}

// JS: invocations (activities) > identifies initial root invocations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L8
func TestActivities_IdentifiesInitialRootInvocations(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
				active.Store(true)
				return nil
			}),
		}},
	})
	xs.CreateActor(machine).Start()

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > identifies initial invocations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L22
func TestActivities_IdentifiesInitialInvocations(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return nil
					}),
				}},
			},
		},
	})
	xs.CreateActor(machine).Start()

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > identifies initial deep invocations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L41
func TestActivities_IdentifiesInitialDeepInvocations(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{
						Key: "a1",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
								active.Store(true)
								return nil
							}),
						}},
					},
				},
			},
		},
	})
	xs.CreateActor(machine).Start()

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > identifies start invocations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L65
func TestActivities_IdentifiesStartInvocations(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"TIMER": {{Target: "b"}}}},
			{
				Key: "b",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return nil
					}),
				}},
			},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("TIMER"))

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > identifies start invocations for child states and active invocations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L92
func TestActivities_IdentifiesStartInvocationsForChildStatesAndActiveInvocations(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"TIMER": {{Target: "b"}}}},
			{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1", On: map[string]xs.Transitions{"TIMER": {{Target: "b2"}}}},
					{
						Key: "b2",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
								active.Store(true)
								return nil
							}),
						}},
					},
				},
			},
		},
	})
	service := xs.CreateActor(machine)

	service.Start()
	service.Send(xs.Ev("TIMER"))
	service.Send(xs.Ev("TIMER"))

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > identifies stop invocations for child states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L130
func TestActivities_IdentifiesStopInvocationsForChildStates(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"TIMER": {{Target: "b"}}}},
			{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1", On: map[string]xs.Transitions{"TIMER": {{Target: "b2"}}}},
					{
						Key: "b2",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
								active.Store(true)
								return func() { active.Store(false) }
							}),
						}},
						On: map[string]xs.Transitions{"TIMER": {{Target: "b3"}}},
					},
					{Key: "b3"},
				},
			},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("TIMER"))
	service.Send(xs.Ev("TIMER"))
	service.Send(xs.Ev("TIMER"))

	assert.Equal(t, false, active.Load())
}

// JS: invocations (activities) > identifies multiple stop invocations for child and parent states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L173
func TestActivities_IdentifiesMultipleStopInvocationsForChildAndParentStates(t *testing.T) {
	var active1, active2 atomic.Bool

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"TIMER": {{Target: "b"}}}},
			{
				Key:     "b",
				Initial: "b1",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active1.Store(true)
						return func() { active1.Store(false) }
					}),
				}},
				States: xs.States{
					{
						Key: "b1",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
								active2.Store(true)
								return func() { active2.Store(false) }
							}),
						}},
					},
				},
				On: map[string]xs.Transitions{"TIMER": {{Target: "a"}}},
			},
		},
	})
	service := xs.CreateActor(machine)

	service.Start()
	service.Send(xs.Ev("TIMER"))
	service.Send(xs.Ev("TIMER"))

	assert.Equal(t, false, active1.Load())
	assert.Equal(t, false, active2.Load())
}

// JS: invocations (activities) > should activate even if there are subsequent always but blocked transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L219
func TestActivities_ShouldActivateEvenIfThereAreSubsequentAlwaysButBlockedTransition(t *testing.T) {
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{"E": {{Target: "B"}}}},
			{
				Key: "B",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return func() { active.Store(false) }
					}),
				}},
				Always: xs.Transitions{{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }),
					Target: "A",
				}},
			},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("E"))

	assert.Equal(t, true, active.Load())
}

// JS: invocations (activities) > should remember the invocations even after an ignored event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L248
func TestActivities_ShouldRememberTheInvocationsEvenAfterAnIgnoredEvent(t *testing.T) {
	cleanupSpy := newSpy()
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{"E": {{Target: "B"}}}},
			{
				Key: "B",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return func() {
							active.Store(false)
							cleanupSpy.Call()
						}
					}),
				}},
			},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("E"))
	service.Send(xs.Ev("IGNORE"))

	assert.Equal(t, true, active.Load())
	assert.Equal(t, 0, cleanupSpy.Count())
}

// JS: invocations (activities) > should remember the invocations when transitioning within the invoking state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L281
func TestActivities_ShouldRememberTheInvocationsWhenTransitioningWithinTheInvokingState(t *testing.T) {
	cleanupSpy := newSpy()
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key: "A",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return func() {
							active.Store(false)
							cleanupSpy.Call()
						}
					}),
				}},
				Initial: "A1",
				States: xs.States{
					{Key: "A1", On: map[string]xs.Transitions{"E": {{Target: "A2"}}}},
					{Key: "A2"},
				},
			},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("E"))

	assert.Equal(t, true, active.Load())
	assert.Equal(t, 0, cleanupSpy.Count())
}

// JS: invocations (activities) > should start a new actor when leaving an invoking state and entering a new one that invokes the same actor type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L317
func TestActivities_ShouldStartNewActorWhenLeavingInvokingStateAndEnteringNewOneInvokingSameActorType(t *testing.T) {
	var counter atomic.Int64
	actual := &activities1Log{}

	fooActor := xs.FromCallback(func(a xs.CallbackArgs) func() {
		localID := counter.Add(1) - 1

		actual.push(fmt.Sprintf("start %d", localID))

		return func() {
			actual.push(fmt.Sprintf("stop %d", localID))
		}
	})

	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"fooActor": fooActor},
	}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:    "a",
				Invoke: []xs.InvokeConfig{{Src: "fooActor"}},
				On:     map[string]xs.Transitions{"NEXT": {{Target: "b"}}},
			},
			{
				Key:    "b",
				Invoke: []xs.InvokeConfig{{Src: "fooActor"}},
			},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{"start 0", "stop 0", "start 1"}, actual.get())
}

// JS: invocations (activities) > should start a new actor when reentering the invoking state during a reentering self transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L361
func TestActivities_ShouldStartNewActorWhenReenteringInvokingStateDuringReenteringSelfTransition(t *testing.T) {
	var counter atomic.Int64
	actual := &activities1Log{}

	fooActor := xs.FromCallback(func(a xs.CallbackArgs) func() {
		localID := counter.Add(1) - 1

		actual.push(fmt.Sprintf("start %d", localID))

		return func() {
			actual.push(fmt.Sprintf("stop %d", localID))
		}
	})

	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"fooActor": fooActor},
	}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:    "a",
				Invoke: []xs.InvokeConfig{{Src: "fooActor"}},
				On: map[string]xs.Transitions{
					"NEXT": {{Target: "a", Reenter: true}},
				},
			},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{"start 0", "stop 0", "start 1"}, actual.get())
}

// JS: invocations (activities) > should have stopped after automatic transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/activities.test.ts#L403
func TestActivities_ShouldHaveStoppedAfterAutomaticTransitions(t *testing.T) {
	type ctx struct{ Counter int }
	var active atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						active.Store(true)
						return func() { active.Store(false) }
					}),
				}},
				Always: xs.Transitions{{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Counter != 0 }),
					Target: "b",
				}},
				On: map[string]xs.Transitions{
					"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Counter: a.Context.Counter + 1}
					})}}},
				},
			},
			{Key: "b"},
		},
	})
	service := xs.CreateActor(machine).Start()

	assert.Equal(t, true, active.Load())

	service.Send(xs.Ev("INC"))

	assert.Equal(t, false, active.Load())
}
