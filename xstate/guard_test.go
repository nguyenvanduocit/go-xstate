package xstate_test

import (
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// guards1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func guards1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
	var mu sync.Mutex
	logs := []string{}

	addTrackingActions := func(state *xs.StateNode, stateDescription string) {
		state.Entry = append(xs.Actions{xs.ActionFunc(func(xs.ActionArgs[C]) {
			mu.Lock()
			defer mu.Unlock()
			logs = append(logs, "enter: "+stateDescription)
		})}, state.Entry...)
		state.Exit = append(xs.Actions{xs.ActionFunc(func(xs.ActionArgs[C]) {
			mu.Lock()
			defer mu.Unlock()
			logs = append(logs, "exit: "+stateDescription)
		})}, state.Exit...)
	}

	var addTrackingActionsRecursively func(state *xs.StateNode)
	addTrackingActionsRecursively = func(state *xs.StateNode) {
		for _, child := range state.ChildStates() {
			addTrackingActions(child, strings.Join(child.Path, "."))
			addTrackingActionsRecursively(child)
		}
	}

	addTrackingActions(machine.RootNode(), "__root__")
	addTrackingActionsRecursively(machine.RootNode())

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		flushed := logs
		logs = []string{}
		return flushed
	}
}

// guards1LightCtx mirrors LightMachineCtx.
type guards1LightCtx struct{ Elapsed int }

// guards1LightMachine mirrors the shared `lightMachine` of the first
// 'guard conditions' describe block. Input: map[string]any{"elapsed": n} or nil.
func guards1LightMachine() *xs.StateMachine[guards1LightCtx] {
	return xs.CreateMachine(xs.MachineConfig[guards1LightCtx]{
		ContextFn: func(a xs.ContextArgs) guards1LightCtx {
			// ({ input = {} }) => ({ elapsed: input.elapsed ?? 0 })
			elapsed := 0
			if in, ok := a.Input.(map[string]any); ok {
				if v, ok := in["elapsed"].(int); ok {
					elapsed = v
				}
			}
			return guards1LightCtx{Elapsed: elapsed}
		},
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {
					{
						Target: "green",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
							return a.Context.Elapsed < 100
						}),
					},
					{
						Target: "yellow",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
							return a.Context.Elapsed >= 100 && a.Context.Elapsed < 200
						}),
					},
				},
				"EMERGENCY": {{
					Target: "red",
					Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
						v, _ := a.Event.(xs.E)["isEmergency"].(bool)
						return v
					}),
				}},
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER":          {{Target: "red", Guard: xs.GuardRef{Type: "minTimeElapsed"}}},
				"TIMER_COND_OBJ": {{Target: "red", Guard: xs.GuardRef{Type: "minTimeElapsed"}}},
			}},
			{Key: "red", On: map[string]xs.Transitions{
				"BAD_COND": {{Target: "red", Guard: xs.GuardRef{Type: "doesNotExist"}}},
			}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"minTimeElapsed": xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
			return a.Context.Elapsed >= 100 && a.Context.Elapsed < 200
		}),
	}})
}

// ---- guard conditions (first block) ----

// JS: guard conditions > should transition only if condition is met
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L81
func TestGuards_GuardConditions_ShouldTransitionOnlyIfConditionIsMet(t *testing.T) {
	lightMachine := guards1LightMachine()

	actorRef1 := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{"elapsed": 50})).Start()
	actorRef1.Send(xs.Ev("TIMER"))
	assert.Equal(t, "green", actorRef1.GetSnapshot().Value)

	actorRef2 := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{"elapsed": 120})).Start()
	actorRef2.Send(xs.Ev("TIMER"))
	assert.Equal(t, "yellow", actorRef2.GetSnapshot().Value)
}

// JS: guard conditions > should transition if condition based on event is met
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L95
func TestGuards_GuardConditions_ShouldTransitionIfConditionBasedOnEventIsMet(t *testing.T) {
	lightMachine := guards1LightMachine()

	actorRef := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{})).Start()
	actorRef.Send(xs.E{"type": "EMERGENCY", "isEmergency": true})
	assert.Equal(t, "red", actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should not transition if condition based on event is not met
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L104
func TestGuards_GuardConditions_ShouldNotTransitionIfConditionBasedOnEventIsNotMet(t *testing.T) {
	lightMachine := guards1LightMachine()

	actorRef := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{})).Start()
	actorRef.Send(xs.Ev("EMERGENCY"))
	assert.Equal(t, "green", actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should not transition if no condition is met
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L112
func TestGuards_GuardConditions_ShouldNotTransitionIfNoConditionIsMet(t *testing.T) {
	eventElapsed := func(e xs.Event) int {
		v, _ := e.(xs.E)["elapsed"].(int)
		return v
	}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"TIMER": {
					{Target: "b", Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return eventElapsed(a.Event) > 200 })},
					{Target: "c", Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return eventElapsed(a.Event) > 100 })},
				},
			}},
			{Key: "b"},
			{Key: "c"},
		},
	})

	flushTracked := guards1TrackEntries(machine)
	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.E{"type": "TIMER", "elapsed": 10})

	assert.Equal(t, "a", actor.GetSnapshot().Value)
	assert.Equal(t, []string{}, flushTracked())
}

// JS: guard conditions > should work with defined string transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L145
func TestGuards_GuardConditions_ShouldWorkWithDefinedStringTransitions(t *testing.T) {
	lightMachine := guards1LightMachine()

	actorRef := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{"elapsed": 120})).Start()
	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, "yellow", actorRef.GetSnapshot().Value)
	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, "red", actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should work with guard objects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L159
func TestGuards_GuardConditions_ShouldWorkWithGuardObjects(t *testing.T) {
	lightMachine := guards1LightMachine()

	actorRef := xs.CreateActor(lightMachine, xs.WithInput(map[string]any{"elapsed": 150})).Start()
	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, "yellow", actorRef.GetSnapshot().Value)
	actorRef.Send(xs.Ev("TIMER_COND_OBJ"))
	assert.Equal(t, "red", actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should work with defined string transitions (condition not met)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L173
func TestGuards_GuardConditions_ShouldWorkWithDefinedStringTransitionsConditionNotMet(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[guards1LightCtx]{
		Context: guards1LightCtx{Elapsed: 10},
		Initial: "yellow",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {
					{
						Target: "green",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
							return a.Context.Elapsed < 100
						}),
					},
					{
						Target: "yellow",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
							return a.Context.Elapsed >= 100 && a.Context.Elapsed < 200
						}),
					},
				},
				"EMERGENCY": {{
					Target: "red",
					Guard: xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
						v, _ := a.Event.(xs.E)["isEmergency"].(bool)
						return v
					}),
				}},
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER": {{Target: "red", Guard: xs.GuardRef{Type: "minTimeElapsed"}}},
			}},
			{Key: "red"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"minTimeElapsed": xs.GuardFunc(func(a xs.GuardArgs[guards1LightCtx]) bool {
			return a.Context.Elapsed >= 100 && a.Context.Elapsed < 200
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("TIMER"))

	assert.Equal(t, "yellow", actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should throw if string transition is not defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L228
func TestGuards_GuardConditions_ShouldThrowIfStringTransitionIsNotDefined(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{
				"BAD_COND": {{Guard: xs.GuardRef{Type: "doesNotExist"}}},
			}},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	actorRef.Send(xs.Ev("BAD_COND"))

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %T", calls[0][0])
	assert.EqualError(t, err,
		"Unable to evaluate guard 'doesNotExist' in transition for event 'BAD_COND' in state node '(machine).foo':\n"+
			"Guard 'doesNotExist' is not implemented.'.")
}

// ---- guard conditions (second block) ----

// JS: guard conditions > should guard against transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L264
func TestGuards_GuardConditions_ShouldGuardAgainstTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A2", States: xs.States{
				{Key: "A0"},
				{Key: "A2"},
			}},
			{Key: "B", Initial: "B0", States: xs.States{
				{
					Key: "B0",
					Always: xs.Transitions{{
						Target: "B4",
						Guard:  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
					}},
					On: map[string]xs.Transitions{
						"T1": {{
							Target: "B1",
							Guard:  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
						}},
					},
				},
				{Key: "B1"},
				{Key: "B4"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("T1"))

	assert.Equal(t, map[string]any{"A": "A2", "B": "B0"}, actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should allow a matching transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L310
func TestGuards_GuardConditions_ShouldAllowAMatchingTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A2", States: xs.States{
				{Key: "A0"},
				{Key: "A2"},
			}},
			{Key: "B", Initial: "B0", States: xs.States{
				{
					Key: "B0",
					Always: xs.Transitions{{
						Target: "B4",
						Guard:  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
					}},
					On: map[string]xs.Transitions{
						"T2": {{Target: "B2", Guard: xs.StateIn("A.A2")}},
					},
				},
				{Key: "B1"},
				{Key: "B2"},
				{Key: "B4"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("T2"))

	assert.Equal(t, map[string]any{"A": "A2", "B": "B2"}, actorRef.GetSnapshot().Value)
}

// JS: guard conditions > should check guards with interim states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L357
func TestGuards_GuardConditions_ShouldCheckGuardsWithInterimStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A2", States: xs.States{
				{Key: "A2", On: map[string]xs.Transitions{"A": {{Target: "A3"}}}},
				{Key: "A3", Always: xs.Transitions{{Target: "A4"}}},
				{Key: "A4", Always: xs.Transitions{{Target: "A5"}}},
				{Key: "A5"},
			}},
			{Key: "B", Initial: "B0", States: xs.States{
				{Key: "B0", Always: xs.Transitions{{Target: "B4", Guard: xs.StateIn("A.A4")}}},
				{Key: "B4"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("A"))

	assert.Equal(t, map[string]any{"A": "A5", "B": "B4"}, actorRef.GetSnapshot().Value)
}

// ---- custom guards ----

// JS: custom guards > should evaluate custom guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L406
func TestGuards_CustomGuards_ShouldEvaluateCustomGuards(t *testing.T) {
	// Ctx is indexed by a dynamic `prop` key, so it is a map.
	type ctx = map[string]int
	type customParams struct {
		Prop    string
		Op      string
		Compare int
	}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "inactive",
		Context: ctx{"count": 0},
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "active",
					Guard: xs.GuardRef{
						Type:   "custom",
						Params: customParams{Prop: "count", Op: "greaterThan", Compare: 3},
					},
				}},
			}},
			{Key: "active"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"custom": xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
			p := a.Params.(customParams)
			if p.Op == "greaterThan" {
				value, _ := a.Event.(xs.E)["value"].(int)
				return a.Context[p.Prop]+value > p.Compare
			}
			return false
		}),
	}})

	actorRef1 := xs.CreateActor(machine).Start()
	actorRef1.Send(xs.E{"type": "EVENT", "value": 4})
	passState := actorRef1.GetSnapshot()

	assert.Equal(t, "active", passState.Value)

	actorRef2 := xs.CreateActor(machine).Start()
	actorRef2.Send(xs.E{"type": "EVENT", "value": 3})
	failState := actorRef2.GetSnapshot()

	assert.Equal(t, "inactive", failState.Value)
}

// JS: custom guards > should provide the undefined params if a guard was configured using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L474
func TestGuards_CustomGuards_ShouldProvideUndefinedParamsIfGuardConfiguredUsingString(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	// toHaveBeenCalledWith(undefined)
	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: custom guards > should provide the guard with resolved params when they are dynamic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L501
func TestGuards_CustomGuards_ShouldProvideGuardWithResolvedParamsWhenDynamic(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{
				Type: "myGuard",
				Params: xs.NewExpr(func(xs.ExprArgs[any]) any {
					return map[string]any{"stuff": 100}
				}),
			}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{map[string]any{"stuff": 100}})
}

// JS: custom guards > should resolve dynamic params using context value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L530
func TestGuards_CustomGuards_ShouldResolveDynamicParamsUsingContextValue(t *testing.T) {
	type ctx struct{ Secret int }
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Secret: 42},
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{
				Type: "myGuard",
				Params: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return map[string]any{"secret": a.Context.Secret}
				}),
			}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{map[string]any{"secret": 42}})
}

// JS: custom guards > should resolve dynamic params using event value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L565
func TestGuards_CustomGuards_ShouldResolveDynamicParamsUsingEventValue(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{
				Type: "myGuard",
				Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
					return map[string]any{"secret": a.Event.(xs.E)["secret"]}
				}),
			}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.E{"type": "FOO", "secret": 77})

	assert.Contains(t, s.Calls(), []any{map[string]any{"secret": 77}})
}

// JS: custom guards > should call a referenced `not` guard that embeds an inline function guard with undefined params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L598
func TestGuards_CustomGuards_ShouldCallReferencedNotEmbeddingInlineGuardWithUndefinedParams(t *testing.T) {
	type ctx struct{ Counter int }
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.Not(xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
			s.Call(a.Params)
			return true
		})),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: custom guards > should call a string guard referenced by referenced `not` with undefined params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L632
func TestGuards_CustomGuards_ShouldCallStringGuardReferencedByReferencedNotWithUndefinedParams(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"other": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
		"myGuard": xs.Not(xs.GuardRef{Type: "other"}),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: custom guards > should call an object guard referenced by referenced `not` with its own params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L664
func TestGuards_CustomGuards_ShouldCallObjectGuardReferencedByReferencedNotWithOwnParams(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"other": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
		"myGuard": xs.Not(xs.GuardRef{Type: "other", Params: 42}),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{42})
}

// JS: custom guards > should call an inline function guard embedded in referenced `and` with undefined params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L699
func TestGuards_CustomGuards_ShouldCallInlineGuardEmbeddedInReferencedAndWithUndefinedParams(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"other": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		"myGuard": xs.And(
			xs.GuardRef{Type: "other"},
			xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
				s.Call(a.Params)
				return true
			}),
		),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: custom guards > should call a string guard referenced by referenced `and` with undefined params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L734
func TestGuards_CustomGuards_ShouldCallStringGuardReferencedByReferencedAndWithUndefinedParams(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"other": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
		"myGuard": xs.And(
			xs.GuardRef{Type: "other"},
			xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: custom guards > should call an object guard referenced by referenced `and` with its own params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L766
func TestGuards_CustomGuards_ShouldCallObjectGuardReferencedByReferencedAndWithOwnParams(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Guard: xs.GuardRef{Type: "myGuard", Params: "foo"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"other": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
		"myGuard": xs.And(
			xs.GuardRef{Type: "other", Params: 42},
			xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		),
	}})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{42})
}

// ---- referencing guards ----

// JS: referencing guards > guard should be checked when referenced by a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L806
func TestGuards_ReferencingGuards_GuardCheckedWhenReferencedByString(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{Guard: xs.GuardRef{Type: "checkStuff"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		// `checkStuff: vi.fn()` returns undefined (falsy).
		"checkStuff": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a, a.Params)
			return false
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, 0, s.Count())

	actorRef.Send(xs.Ev("EV"))

	assert.Equal(t, 1, s.Count())
}

// JS: referencing guards > guard should be checked when referenced by a parametrized guard object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L834
func TestGuards_ReferencingGuards_GuardCheckedWhenReferencedByParametrizedGuardObject(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{Guard: xs.GuardRef{Type: "checkStuff"}}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		// `checkStuff: vi.fn()` returns undefined (falsy).
		"checkStuff": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a, a.Params)
			return false
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, 0, s.Count())

	actorRef.Send(xs.Ev("EV"))

	assert.Equal(t, 1, s.Count())
}

// JS: referencing guards > should throw for guards with missing predicates
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L864
func TestGuards_ReferencingGuards_ShouldThrowForGuardsWithMissingPredicates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "invalid-predicate",
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"EVENT": {{Target: "inactive", Guard: xs.GuardRef{Type: "missing-predicate"}}},
			}},
			{Key: "inactive"},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()
	actorRef.Send(xs.Ev("EVENT"))

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %T", calls[0][0])
	assert.EqualError(t, err,
		"Unable to evaluate guard 'missing-predicate' in transition for event 'EVENT' in state node 'invalid-predicate.active':\n"+
			"Guard 'missing-predicate' is not implemented.'.")
}

// JS: referencing guards > should be possible to reference a composite guard that only uses inline predicates
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L897
func TestGuards_ReferencingGuards_ReferenceCompositeGuardWithOnlyInlinePredicates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{Target: "b", Guard: xs.GuardRef{Type: "referenced"}}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"referenced": xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false })),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: referencing guards > should be possible to reference a composite guard that references other guards recursively
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L926
func TestGuards_ReferencingGuards_ReferenceCompositeGuardReferencingOtherGuardsRecursively(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{Target: "b", Guard: xs.GuardRef{Type: "referenced"}}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		"falsy":  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
		"referenced": xs.Or(
			xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
			xs.Not(xs.GuardRef{Type: "truthy"}),
			xs.And(xs.Not(xs.GuardRef{Type: "falsy"}), xs.GuardRef{Type: "truthy"}),
		),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: referencing guards > should be possible to resolve referenced guards recursively
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L961
func TestGuards_ReferencingGuards_ResolveReferencedGuardsRecursively(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{Target: "b", Guard: xs.GuardRef{Type: "ref1"}}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"ref1": xs.GuardRef{Type: "ref2"},
		"ref2": xs.GuardRef{Type: "ref3"},
		"ref3": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// ---- guards - other ----

// JS: guards - other > should allow for a fallback target to be a simple string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L994
func TestGuards_Other_ShouldAllowFallbackTargetToBeSimpleString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {
					{Target: "b", Guard: xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false })},
					{Target: "c"},
				},
			}},
			{Key: "b"},
			{Key: "c"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("EVENT"))

	assert.Equal(t, "c", service.GetSnapshot().Value)
}

// JS: guards - other > inline function guard should not leak into provided guards object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1014
func TestGuards_Other_InlineFunctionGuardShouldNotLeakIntoProvidedGuards(t *testing.T) {
	guards := map[string]xs.Guard{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{
				Guard:   xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
				Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{Guards: guards})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, map[string]xs.Guard{}, guards)
}

// JS: guards - other > inline builtin guard should not leak into provided guards object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1035
func TestGuards_Other_InlineBuiltinGuardShouldNotLeakIntoProvidedGuards(t *testing.T) {
	guards := map[string]xs.Guard{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{
				Guard:   xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false })),
				Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{Guards: guards})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, map[string]xs.Guard{}, guards)
}

// ---- not() guard ----

// JS: not() guard > should guard with inline function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1058
func TestGuards_Not_ShouldGuardWithInlineFunction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard:  xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false })),
				}},
			}},
			{Key: "b"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: not() guard > should guard with string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1081
func TestGuards_Not_ShouldGuardWithString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{Target: "b", Guard: xs.Not(xs.GuardRef{Type: "falsy"})}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"falsy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: not() guard > should guard with object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1110
func TestGuards_Not_ShouldGuardWithObject(t *testing.T) {
	type greaterThan10Params struct{ Value int }

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard:  xs.Not(xs.GuardRef{Type: "greaterThan10", Params: greaterThan10Params{Value: 5}}),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"greaterThan10": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			return a.Params.(greaterThan10Params).Value > 10
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: not() guard > should guard with nested built-in guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1144
func TestGuards_Not_ShouldGuardWithNestedBuiltInGuards(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.Not(xs.And(
						xs.Not(xs.GuardRef{Type: "truthy"}),
						xs.GuardRef{Type: "truthy"},
					)),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		"falsy":  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: not() guard > should evaluate dynamic params of the referenced guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1174
func TestGuards_Not_ShouldEvaluateDynamicParamsOfReferencedGuard(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{
				Guard: xs.Not(xs.GuardRef{
					Type: "myGuard",
					Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
						return map[string]any{"secret": a.Event.(xs.E)["secret"]}
					}),
				}),
				Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.E{"type": "EV", "secret": 42})

	assert.Equal(t, [][]any{{map[string]any{"secret": 42}}}, s.Calls())
}

// ---- and() guard ----

// JS: and() guard > should guard with inline function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1216
func TestGuards_And_ShouldGuardWithInlineFunction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.And(
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return 1+1 == 2 }),
					),
				}},
			}},
			{Key: "b"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: and() guard > should guard with string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1238
func TestGuards_And_ShouldGuardWithString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard:  xs.And(xs.GuardRef{Type: "truthy"}, xs.GuardRef{Type: "truthy"}),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: and() guard > should guard with object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1267
func TestGuards_And_ShouldGuardWithObject(t *testing.T) {
	type greaterThan10Params struct{ Value int }

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.And(
						xs.GuardRef{Type: "greaterThan10", Params: greaterThan10Params{Value: 11}},
						xs.GuardRef{Type: "greaterThan10", Params: greaterThan10Params{Value: 50}},
					),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"greaterThan10": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			return a.Params.(greaterThan10Params).Value > 10
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: and() guard > should guard with nested built-in guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1307
func TestGuards_And_ShouldGuardWithNestedBuiltInGuards(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.And(
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
						xs.Not(xs.GuardRef{Type: "falsy"}),
						xs.And(xs.Not(xs.GuardRef{Type: "falsy"}), xs.GuardRef{Type: "truthy"}),
					),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		"falsy":  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: and() guard > should evaluate dynamic params of the referenced guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1341
func TestGuards_And_ShouldEvaluateDynamicParamsOfReferencedGuard(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{
				Guard: xs.And(
					xs.GuardRef{
						Type: "myGuard",
						Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
							return map[string]any{"secret": a.Event.(xs.E)["secret"]}
						}),
					},
					xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				),
				Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.E{"type": "EV", "secret": 42})

	assert.Equal(t, [][]any{{map[string]any{"secret": 42}}}, s.Calls())
}

// ---- or() guard ----

// JS: or() guard > should guard with inline function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1386
func TestGuards_Or_ShouldGuardWithInlineFunction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.Or(
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return 1+1 == 2 }),
					),
				}},
			}},
			{Key: "b"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: or() guard > should guard with string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1408
func TestGuards_Or_ShouldGuardWithString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard:  xs.Or(xs.GuardRef{Type: "falsy"}, xs.GuardRef{Type: "truthy"}),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"falsy":  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: or() guard > should guard with object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1438
func TestGuards_Or_ShouldGuardWithObject(t *testing.T) {
	type greaterThan10Params struct{ Value int }

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.Or(
						xs.GuardRef{Type: "greaterThan10", Params: greaterThan10Params{Value: 4}},
						xs.GuardRef{Type: "greaterThan10", Params: greaterThan10Params{Value: 50}},
					),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"greaterThan10": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			return a.Params.(greaterThan10Params).Value > 10
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: or() guard > should guard with nested built-in guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1478
func TestGuards_Or_ShouldGuardWithNestedBuiltInGuards(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{
					Target: "b",
					Guard: xs.Or(
						xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
						xs.Not(xs.GuardRef{Type: "truthy"}),
						xs.And(xs.Not(xs.GuardRef{Type: "falsy"}), xs.GuardRef{Type: "truthy"}),
					),
				}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"truthy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
		"falsy":  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.True(t, actorRef.GetSnapshot().Matches("b"))
}

// JS: or() guard > should evaluate dynamic params of the referenced guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/guards.test.ts#L1512
func TestGuards_Or_ShouldEvaluateDynamicParamsOfReferencedGuard(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{
				Guard: xs.Or(
					xs.GuardRef{
						Type: "myGuard",
						Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
							return map[string]any{"secret": a.Event.(xs.E)["secret"]}
						}),
					},
					xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				),
				Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"myGuard": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
			s.Call(a.Params)
			return true
		}),
	}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.E{"type": "EV", "secret": 42})

	assert.Equal(t, [][]any{{map[string]any{"secret": 42}}}, s.Calls())
}
