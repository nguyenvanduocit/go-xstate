package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: select > should get current value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L6
func TestSelect_ShouldGetCurrentValue(t *testing.T) {
	type ctx struct{ Data int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: 42},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Data = a.Context.Data + 1
					return c
				})}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()
	selection := xs.Select(service, func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Data })

	assert.Equal(t, 42, selection.Get())

	service.Send(xs.Ev("INC"))

	assert.Equal(t, 43, selection.Get())
}

// JS: select > should subscribe to changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L32
func TestSelect_ShouldSubscribeToChanges(t *testing.T) {
	type ctx struct{ Data int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: 42},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Data = a.Context.Data + 1
					return c
				})}}},
			}},
		},
	})

	callback := newSpy()
	service := xs.CreateActor(machine).Start()
	selection := xs.Select(service, func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Data })
	selection.Subscribe(xs.Observer[int]{Next: func(v int) { callback.Call(v) }})

	service.Send(xs.Ev("INC"))

	assert.Equal(t, 1, callback.Count())
	assert.Contains(t, callback.Calls(), []any{43})
}

// JS: select > should not notify if selected value has not changed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L59
func TestSelect_ShouldNotNotifyIfSelectedValueHasNotChanged(t *testing.T) {
	type ctx struct {
		Data  int
		Other string
	}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: 42, Other: "foo"},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Data = a.Context.Data + 1
					return c
				})}}},
			}},
		},
	})

	callback := newSpy()
	service := xs.CreateActor(machine).Start()
	selection := xs.Select(service, func(s *xs.MachineSnapshot[ctx]) string { return s.Context.Other })
	selection.Subscribe(xs.Observer[string]{Next: func(v string) { callback.Call(v) }})

	service.Send(xs.Ev("INC"))

	assert.Equal(t, 0, callback.Count())
}

// JS: select > should support custom equality function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L85
func TestSelect_ShouldSupportCustomEqualityFunction(t *testing.T) {
	type ctx struct {
		Age  int
		Name string
	}
	type selected struct {
		Name string
		Age  int
	}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Age: 42, Name: "John"},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"UPDATE_NAME": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Name = a.Event.(xs.E)["name"].(string)
					return c
				})}}},
				"UPDATE_AGE": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Age = a.Event.(xs.E)["age"].(int)
					return c
				})}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	callback := newSpy()
	selector := func(s *xs.MachineSnapshot[ctx]) selected {
		return selected{Name: s.Context.Name, Age: s.Context.Age}
	}
	equalityFn := func(a, b selected) bool {
		return a.Name == b.Name // Only compare names
	}

	xs.Select(service, selector, equalityFn).Subscribe(xs.Observer[selected]{
		Next: func(v selected) { callback.Call(v) },
	})

	service.Send(xs.E{"type": "UPDATE_AGE", "age": 66})
	assert.Equal(t, 0, callback.Count())

	service.Send(xs.E{"type": "UPDATE_NAME", "name": "Jane"})
	assert.Equal(t, 1, callback.Count())
}

// JS: select > should unsubscribe correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L134
func TestSelect_ShouldUnsubscribeCorrectly(t *testing.T) {
	type ctx struct{ Data int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: 42},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Data = a.Context.Data + 1
					return c
				})}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	callback := newSpy()
	selection := xs.Select(service, func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Data })
	subscription := selection.Subscribe(xs.Observer[int]{Next: func(v int) { callback.Call(v) }})

	subscription.Unsubscribe()
	service.Send(xs.Ev("INC"))

	assert.Equal(t, 0, callback.Count())
}

// JS: select > should handle updates with multiple subscribers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/select.test.ts#L162
func TestSelect_ShouldHandleUpdatesWithMultipleSubscribers(t *testing.T) {
	type position struct{ X, Y int }
	type user struct {
		Age  int
		Name string
	}
	type ctx struct {
		User     user
		Position position
	}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Position: position{X: 0, Y: 0}, User: user{Name: "John", Age: 30}},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{
				"UPDATE_USER": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.User = a.Event.(xs.E)["user"].(user)
					return c
				})}}},
				"UPDATE_POSITION": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Position = a.Event.(xs.E)["position"].(position)
					return c
				})}}},
			}},
		},
	})

	store := xs.CreateActor(machine).Start()

	// Mock DOM manipulation callback
	renderCallback := newSpy()
	xs.Select(store, func(s *xs.MachineSnapshot[ctx]) position { return s.Context.Position }).
		Subscribe(xs.Observer[position]{Next: func(p position) {
			renderCallback.Call(p)
		}})

	// Mock logger callback for x position only
	loggerCallback := newSpy()
	xs.Select(store, func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Position.X }).
		Subscribe(xs.Observer[int]{Next: func(x int) {
			loggerCallback.Call(x)
		}})

	lastCall := func(s *spy) []any {
		calls := s.Calls()
		if len(calls) == 0 {
			return nil
		}
		return calls[len(calls)-1]
	}

	// Simulate position update
	store.Send(xs.E{"type": "UPDATE_POSITION", "position": position{X: 100, Y: 200}})

	// Verify render callback received full position update
	assert.Equal(t, 1, renderCallback.Count())
	assert.Contains(t, renderCallback.Calls(), []any{position{X: 100, Y: 200}})

	// Verify logger callback received only x position
	assert.Equal(t, 1, loggerCallback.Count())
	assert.Contains(t, loggerCallback.Calls(), []any{100})

	// Simulate another update
	store.Send(xs.E{"type": "UPDATE_POSITION", "position": position{X: 150, Y: 300}})

	assert.Equal(t, 2, renderCallback.Count())
	assert.Equal(t, []any{position{X: 150, Y: 300}}, lastCall(renderCallback))
	assert.Equal(t, 2, loggerCallback.Count())
	assert.Equal(t, []any{150}, lastCall(loggerCallback))

	// Simulate changing only the y position
	store.Send(xs.E{"type": "UPDATE_POSITION", "position": position{X: 150, Y: 400}})

	assert.Equal(t, 3, renderCallback.Count())
	assert.Equal(t, []any{position{X: 150, Y: 400}}, lastCall(renderCallback))

	// loggerCallback should not have been called
	assert.Equal(t, 2, loggerCallback.Count())

	// Simulate changing only the user
	store.Send(xs.E{"type": "UPDATE_USER", "user": user{Name: "Jane", Age: 25}})

	// renderCallback should not have been called
	assert.Equal(t, 3, renderCallback.Count())

	// loggerCallback should not have been called
	assert.Equal(t, 2, loggerCallback.Count())
}
