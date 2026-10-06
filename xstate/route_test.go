package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// route1Ev mirrors `{ type: 'xstate.route', to }`.
func route1Ev(to string) xs.E { return xs.E{"type": "xstate.route", "to": to} }

// JS: route > should transition directly to a route if route is an empty transition config
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L4
func TestRoute_ShouldTransitionDirectlyToRouteIfRouteIsEmptyTransitionConfig(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b", ID: "b", Route: &xs.TransitionConfig{}},
			{Key: "c"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(route1Ev("#b"))

	assert.Equal(t, "b", actor.GetSnapshot().Value)

	// c has no route, so this should not transition
	actor.Send(route1Ev("#c"))

	assert.Equal(t, "b", actor.GetSnapshot().Value)
}

// JS: route > should transition directly to a route if guard passes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L36
func TestRoute_ShouldTransitionDirectlyToRouteIfGuardPasses(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b", ID: "b", Route: &xs.TransitionConfig{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }),
			}},
			{Key: "c", ID: "c", Route: &xs.TransitionConfig{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }),
			}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, "a", actor.GetSnapshot().Value)

	actor.Send(route1Ev("#b"))

	assert.Equal(t, "a", actor.GetSnapshot().Value)

	actor.Send(route1Ev("#c"))

	assert.Equal(t, "c", actor.GetSnapshot().Value)
}

// JS: route > should resolve setup-registered string guards on route transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L76
func TestRoute_ShouldResolveSetupRegisteredStringGuardsOnRouteTransitions(t *testing.T) {
	type ctx struct{ Ready bool }

	machine := xs.NewSetup[ctx](xs.Implementations{
		Guards: map[string]xs.Guard{
			"isReady": xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Ready }),
		},
	}).CreateMachine(xs.MachineConfig[ctx]{
		ID:      "flow",
		Initial: "amount",
		Context: ctx{Ready: false},
		States: xs.States{
			{
				Key:   "amount",
				ID:    "amount",
				Route: &xs.TransitionConfig{},
				On: map[string]xs.Transitions{
					"READY": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Ready = true
						return c
					})}}},
				},
			},
			{
				Key:   "review",
				ID:    "review",
				Route: &xs.TransitionConfig{Guard: xs.GuardRef{Type: "isReady"}},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(route1Ev("#review"))

	assert.Equal(t, "amount", actor.GetSnapshot().Value)

	actor.Send(xs.Ev("READY"))
	actor.Send(route1Ev("#review"))

	assert.Equal(t, "review", actor.GetSnapshot().Value)
}

// JS: route > should work with parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L127
func TestRoute_ShouldWorkWithParallelStates(t *testing.T) {
	todoMachine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:   "todos",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "todo",
				Initial: "new",
				States: xs.States{
					{Key: "new"},
					{Key: "editing"},
				},
			},
			{
				Key:     "filter",
				Initial: "all",
				States: xs.States{
					{Key: "all", ID: "filter-all", Route: &xs.TransitionConfig{}},
					{Key: "active", ID: "filter-active", Route: &xs.TransitionConfig{}},
					{Key: "completed", ID: "filter-completed", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	todoActor := xs.CreateActor(todoMachine).Start()

	assert.Equal(t, map[string]any{
		"todo":   "new",
		"filter": "all",
	}, todoActor.GetSnapshot().Value)

	todoActor.Send(route1Ev("#filter-active"))

	assert.Equal(t, map[string]any{
		"todo":   "new",
		"filter": "active",
	}, todoActor.GetSnapshot().Value)
}

// JS: route > route events are strongly typed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L177
func TestRoute_RouteEventsAreStronglyTyped(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "aRoute",
		States: xs.States{
			{Key: "aRoute", ID: "aRoute", Route: &xs.TransitionConfig{}},
			{
				Key:     "notARoute",
				Initial: "childRoute",
				States: xs.States{
					{Key: "childRoute", ID: "childRoute", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(route1Ev("#aRoute"))
	// Added (JS has no expect): valid route to the initial state.
	assert.Equal(t, "aRoute", actor.GetSnapshot().Value)

	actor.Send(route1Ev("#childRoute"))
	// Added (JS has no expect): valid route to a nested route.
	assert.Equal(t, map[string]any{"notARoute": "childRoute"}, actor.GetSnapshot().Value)

	// The following targets are compile errors in TS (@ts-expect-error); at runtime
	// they are not routable and must be ignored without panicking.
	actor.Send(route1Ev("notARoute")) // 'notARoute' has no route config
	actor.Send(route1Ev("root"))      // 'root' is not routable
	actor.Send(route1Ev("blahblah"))  // 'blahblah' does not exist

	// Added (JS has no expect): invalid targets leave the snapshot unchanged.
	assert.Equal(t, map[string]any{"notARoute": "childRoute"}, actor.GetSnapshot().Value)
}

// JS: route > route config without id should not generate route events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L233
func TestRoute_RouteConfigWithoutIDShouldNotGenerateRouteEvents(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			// route without id — should NOT be routable
			{Key: "a", Route: &xs.TransitionConfig{}},
			{Key: "b", ID: "b", Route: &xs.TransitionConfig{}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	// Only 'b' should be a valid route target
	actor.Send(route1Ev("#b"))

	assert.Equal(t, "b", actor.GetSnapshot().Value)
}

// JS: route > machine.root.on should include route events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L264
func TestRoute_MachineRootOnShouldIncludeRouteEvents(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b", ID: "b", Route: &xs.TransitionConfig{}},
			{Key: "c", ID: "c", Route: &xs.TransitionConfig{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }),
			}},
		},
	})

	assert.Contains(t, machine.Root.On(), "xstate.route")
}

// JS: route > nested state on should include route events for child routes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L286
func TestRoute_NestedStateOnShouldIncludeRouteEventsForChildRoutes(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "app",
		Initial: "home",
		States: xs.States{
			{Key: "home", ID: "home", Route: &xs.TransitionConfig{}},
			{
				Key:     "dashboard",
				ID:      "dashboard",
				Initial: "overview",
				Route:   &xs.TransitionConfig{},
				States: xs.States{
					{Key: "overview", ID: "overview", Route: &xs.TransitionConfig{}},
					{Key: "settings", ID: "settings", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	a := xs.CreateActor(machine).Start()
	a.Send(route1Ev("#overview"))

	assert.Equal(t, map[string]any{"dashboard": "overview"}, a.GetSnapshot().Value)

	// All routes should be accessible via 'xstate.route'
	assert.Contains(t, machine.Root.On(), "xstate.route")
}

// JS: route > parallel state on should include route events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L325
func TestRoute_ParallelStateOnShouldIncludeRouteEvents(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:   "todos",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "list",
				Initial: "idle",
				States: xs.States{
					{Key: "idle"},
					{Key: "loading"},
				},
			},
			{
				Key:     "filter",
				Initial: "all",
				States: xs.States{
					{Key: "all", ID: "filter-all", Route: &xs.TransitionConfig{}},
					{Key: "active", ID: "filter-active", Route: &xs.TransitionConfig{}},
					{Key: "completed", ID: "filter-completed", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	// Routes should be accessible
	assert.Contains(t, machine.Root.On(), "xstate.route")
}

// JS: route > should route to deeply nested state from anywhere
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L361
func TestRoute_ShouldRouteToDeeplyNestedStateFromAnywhere(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "app",
		Initial: "home",
		States: xs.States{
			{Key: "home", ID: "home", Route: &xs.TransitionConfig{}},
			{
				Key:     "dashboard",
				Initial: "overview",
				States: xs.States{
					{Key: "overview", ID: "overview", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	// Should be able to route to deeply nested state from root
	assert.Equal(t, "home", actor.GetSnapshot().Value)

	actor.Send(route1Ev("#overview"))

	assert.Equal(t, map[string]any{"dashboard": "overview"}, actor.GetSnapshot().Value)
}

// JS: route > should re-enter when routing to the current state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L392
func TestRoute_ShouldReEnterWhenRoutingToCurrentState(t *testing.T) {
	entries := 0
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				ID:    "a",
				Route: &xs.TransitionConfig{},
				Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					entries++
				})},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()
	assert.Equal(t, "a", actor.GetSnapshot().Value)
	entries = 0

	actor.Send(route1Ev("#a"))

	assert.Equal(t, "a", actor.GetSnapshot().Value)
	assert.Equal(t, 1, entries)
}

// JS: route > should route to self with guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L418
func TestRoute_ShouldRouteToSelfWithGuard(t *testing.T) {
	allowed := false
	entries := 0
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				ID:  "a",
				Route: &xs.TransitionConfig{
					Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return allowed }),
				},
				Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					entries++
				})},
			},
			{Key: "b", ID: "b", Route: &xs.TransitionConfig{}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	entries = 0

	actor.Send(route1Ev("#a"))
	assert.Equal(t, 0, entries)

	allowed = true
	actor.Send(route1Ev("#a"))
	assert.Equal(t, 1, entries)
}

// JS: route > should not route using dot-separated nested id like #id.nested
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/route.test.ts#L449
func TestRoute_ShouldNotRouteUsingDotSeparatedNestedIDLikeIDNested(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		ID:      "app",
		Initial: "home",
		States: xs.States{
			{Key: "home", ID: "home", Route: &xs.TransitionConfig{}},
			{
				Key:     "dashboard",
				ID:      "dashboard",
				Initial: "overview",
				Route:   &xs.TransitionConfig{},
				States: xs.States{
					{Key: "overview", ID: "overview", Route: &xs.TransitionConfig{}},
				},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, "home", actor.GetSnapshot().Value)

	// Dot-separated ids should not work as route targets
	actor.Send(route1Ev("#dashboard.overview"))

	assert.Equal(t, "home", actor.GetSnapshot().Value)
}
