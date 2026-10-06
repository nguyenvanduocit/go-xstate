package xstate_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invoke1User mirrors the top-level `const user = { name: 'David' }`.
type invoke1User struct{ Name string }

var invoke1UserData = invoke1User{Name: "David"}

// invoke1DoneOutput returns event.output of an "xstate.done.actor.*" event.
func invoke1DoneOutput(e xs.Event) any {
	ev, ok := e.(xs.DoneActorEvent)
	if !ok {
		return nil
	}
	return ev.Output
}

// invoke1OutputField reads event.output[key] of an "xstate.done.actor.*" event
// whose output is a JS object literal.
func invoke1OutputField(e xs.Event, key string) any {
	m, ok := invoke1DoneOutput(e).(map[string]any)
	if !ok {
		return nil
	}
	return m[key]
}

// ---- promise factories (JS: `const promiseTypes = [...]`) ----

// invoke1Executor mirrors the JS PromiseExecutor `(resolve, reject) => void`.
type invoke1Executor func(resolve func(value any), reject func(reason any))

// invoke1PromiseType mirrors one entry of `promiseTypes`: createPromise builds
// the promise from executor and awaits it (the body of the fromPromise
// creator), so its result is the (output, error) pair FromPromise expects.
type invoke1PromiseType struct {
	name          string
	createPromise func(ctx context.Context, executor invoke1Executor) (any, error)
}

// invoke1Rejection carries a non-Error rejection reason (`reject()` /
// `reject(value)` / `throw value`) as a Go error.
type invoke1Rejection struct{ reason any }

func (r invoke1Rejection) Error() string { return fmt.Sprint(r.reason) }

func invoke1ToError(reason any) error {
	if err, ok := reason.(error); ok {
		return err
	}
	return invoke1Rejection{reason: reason}
}

type invoke1Settled struct {
	value any
	err   error
}

// invoke1RunExecutor mirrors `new Promise(executor)`: the executor runs
// synchronously, the first resolve/reject wins and a throw rejects.
func invoke1RunExecutor(executor invoke1Executor) <-chan invoke1Settled {
	ch := make(chan invoke1Settled, 1)
	var once sync.Once
	settle := func(s invoke1Settled) { once.Do(func() { ch <- s }) }
	func() {
		defer func() {
			if r := recover(); r != nil {
				settle(invoke1Settled{err: invoke1ToError(r)})
			}
		}()
		executor(
			func(v any) { settle(invoke1Settled{value: v}) },
			func(r any) { settle(invoke1Settled{err: invoke1ToError(r)}) },
		)
	}()
	return ch
}

func invoke1Await(ctx context.Context, ch <-chan invoke1Settled) (any, error) {
	select {
	case s := <-ch:
		return s.value, s.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var invoke1PromiseTypes = map[string]invoke1PromiseType{
	"Promise": {
		name: "Promise",
		createPromise: func(ctx context.Context, executor invoke1Executor) (any, error) {
			return invoke1Await(ctx, invoke1RunExecutor(executor))
		},
	},
	"PromiseLike": {
		name: "PromiseLike",
		// Simulates a Promise/A+ thenable: the settlement of the native promise
		// is relayed through a `then(onfulfilled, onrejected)` hop.
		createPromise: func(ctx context.Context, executor invoke1Executor) (any, error) {
			native := invoke1RunExecutor(executor)
			thenable := make(chan invoke1Settled, 1)
			go func() {
				v, err := invoke1Await(ctx, native)
				thenable <- invoke1Settled{value: v, err: err}
			}()
			return invoke1Await(ctx, thenable)
		},
	},
}

// ---- describe('invoke') ----

// JS: invoke > child can immediately respond to the parent with multiple events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L31
func TestInvoke_ChildCanImmediatelyRespondToTheParentWithMultipleEvents(t *testing.T) {
	type ctx struct{ Count int }

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "init",
		States: xs.States{
			{Key: "init", On: map[string]xs.Transitions{
				"FORWARD_DEC": {{Actions: xs.Actions{
					xs.SendParent(xs.Ev("DEC")),
					xs.SendParent(xs.Ev("DEC")),
					xs.SendParent(xs.Ev("DEC")),
				}}},
			}},
		},
	})

	someParentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "parent",
		Context: ctx{Count: 0},
		Initial: "start",
		States: xs.States{
			{
				Key:    "start",
				Invoke: []xs.InvokeConfig{{Src: "child", ID: "someService"}},
				Always: xs.Transitions{{
					Target: "stop",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count == -3 }),
				}},
				On: map[string]xs.Transitions{
					"DEC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Count = a.Context.Count - 1
						return c
					})}}},
					"FORWARD_DEC": {{Actions: xs.Actions{xs.SendTo("someService", xs.Ev("FORWARD_DEC"))}}},
				},
			},
			{Key: "stop", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": childMachine}})

	actorRef := xs.CreateActor(someParentMachine).Start()
	actorRef.Send(xs.Ev("FORWARD_DEC"))

	// 1. The 'parent' machine will not do anything (inert transition)
	// 2. The 'FORWARD_DEC' event will be "forwarded" to the child machine
	// 3. On the child machine, the 'FORWARD_DEC' event sends the 'DEC' action to the parent thrice
	// 4. The context of the 'parent' machine will be updated from 0 to -3
	assert.Equal(t, ctx{Count: -3}, actorRef.GetSnapshot().Context)
}

// JS: invoke > should start services (explicit machine, invoke = config)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L107
func TestInvoke_ShouldStartServicesExplicitMachineInvokeConfig(t *testing.T) {
	type childCtx struct {
		UserID *string // undefined when nil
		User   any     // undefined when nil
	}
	type parentCtx struct {
		SelectedUserID string
		User           any
	}

	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[childCtx]{
		ID: "fetch",
		ContextFn: func(a xs.ContextArgs) childCtx {
			in, _ := a.Input.(map[string]any)
			var c childCtx
			if id, ok := in["userId"].(string); ok {
				c.UserID = &id
			}
			return c
		},
		Initial: "pending",
		States: xs.States{
			{
				Key:   "pending",
				Entry: xs.Actions{xs.Raise(xs.E{"type": "RESOLVE", "user": invoke1UserData})},
				On: map[string]xs.Transitions{
					"RESOLVE": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[childCtx]) bool {
							return a.Context.UserID != nil
						}),
					}},
				},
			},
			{
				Key:  "success",
				Type: xs.Final,
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[childCtx]) childCtx {
					c := a.Context
					c.User = a.Event.(xs.E)["user"]
					return c
				})},
			},
			{
				Key:   "failure",
				Entry: xs.Actions{xs.SendParent(xs.Ev("REJECT"))},
			},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[childCtx]) any {
			return map[string]any{"user": a.Context.User}
		}),
	})

	machine := xs.CreateMachine(xs.MachineConfig[parentCtx]{
		ID:      "fetcher",
		Initial: "idle",
		Context: parentCtx{SelectedUserID: "42", User: nil},
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"GO_TO_WAITING": {{Target: "waiting"}}}},
			{
				Key: "waiting",
				Invoke: []xs.InvokeConfig{{
					Logic: childMachine,
					Input: xs.NewExpr(func(a xs.ExprArgs[parentCtx]) any {
						return map[string]any{"userId": a.Context.SelectedUserID}
					}),
					OnDone: xs.Transitions{{
						Target: "received",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[parentCtx]) bool {
							// Should receive { user: { name: 'David' } } as event data
							u, ok := invoke1OutputField(a.Event, "user").(invoke1User)
							return ok && u.Name == "David"
						}),
					}},
				}},
			},
			{Key: "received", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[parentCtx]]{
		Complete: func() { sig.Resolve() },
	})
	actor.Start()
	actor.Send(xs.Ev("GO_TO_WAITING"))
	sig.Wait(t)
}

// JS: invoke > should start services (explicit machine, invoke = machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L199
func TestInvoke_ShouldStartServicesExplicitMachineInvokeMachine(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{
				Key:   "pending",
				Entry: xs.Actions{xs.Raise(xs.Ev("RESOLVE"))},
				On:    map[string]xs.Transitions{"RESOLVE": {{Target: "success"}}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"GO_TO_WAITING": {{Target: "waiting"}}}},
			{
				Key: "waiting",
				Invoke: []xs.InvokeConfig{{
					Logic:  childMachine,
					OnDone: xs.Transitions{{Target: "received"}},
				}},
			},
			{Key: "received", Type: xs.Final},
		},
	})
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { sig.Resolve() },
	})
	actor.Start()
	actor.Send(xs.Ev("GO_TO_WAITING"))
	sig.Wait(t)
}

// JS: invoke > should start services (machine as invoke config)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L252
func TestInvoke_ShouldStartServicesMachineAsInvokeConfig(t *testing.T) {
	sig := newSignal()
	machineInvokeMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "machine-invoke",
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.CreateMachine(xs.MachineConfig[any]{
						ID:      "child",
						Initial: "sending",
						States: xs.States{
							{Key: "sending", Entry: xs.Actions{xs.SendParent(xs.E{"type": "SUCCESS", "data": 42})}},
						},
					}),
				}},
				On: map[string]xs.Transitions{
					"SUCCESS": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							return a.Event.(xs.E)["data"] == 42
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})
	actor := xs.CreateActor(machineInvokeMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > should start deeply nested service (machine as invoke config)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L296
func TestInvoke_ShouldStartDeeplyNestedServiceMachineAsInvokeConfig(t *testing.T) {
	sig := newSignal()
	machineInvokeMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "b",
				States: xs.States{
					{
						Key: "b",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.CreateMachine(xs.MachineConfig[any]{
								ID:      "child",
								Initial: "sending",
								States: xs.States{
									{Key: "sending", Entry: xs.Actions{xs.SendParent(xs.E{"type": "SUCCESS", "data": 42})}},
								},
							}),
						}},
					},
				},
			},
			{Key: "success", ID: "success", Type: xs.Final},
		},
		On: map[string]xs.Transitions{
			"SUCCESS": {{
				Target: ".success",
				Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
					return a.Event.(xs.E)["data"] == 42
				}),
			}},
		},
	})
	actor := xs.CreateActor(machineInvokeMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > should use the service overwritten by .provide(...)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L346
func TestInvoke_ShouldUseTheServiceOverwrittenByProvide(t *testing.T) {
	type ctx struct{ Count int }

	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "init",
		States:  xs.States{{Key: "init"}},
	})

	someParentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "parent",
		Context: ctx{Count: 0},
		Initial: "start",
		States: xs.States{
			{
				Key:    "start",
				Invoke: []xs.InvokeConfig{{Src: "child", ID: "someService"}},
				On:     map[string]xs.Transitions{"STOP": {{Target: "stop"}}},
			},
			{Key: "stop", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": childMachine}})

	actor := xs.CreateActor(someParentMachine.Provide(xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"child": xs.CreateMachine(xs.MachineConfig[any]{
				ID:      "child",
				Initial: "init",
				States: xs.States{
					{Key: "init", Entry: xs.Actions{xs.SendParent(xs.Ev("STOP"))}},
				},
			}),
		},
	}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// ---- describe('parent to child') ----

// invoke1ParentToChildSubMachine mirrors `subMachine` declared in the
// 'parent to child' describe body.
func invoke1ParentToChildSubMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "one",
		States: xs.States{
			{Key: "one", On: map[string]xs.Transitions{"NEXT": {{Target: "two"}}}},
			{Key: "two", Entry: xs.Actions{xs.SendParent(xs.Ev("NEXT"))}},
		},
	})
}

// JS: invoke > parent to child > should communicate with the child machine (invoke on machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L429
func TestInvoke_ParentToChild_ShouldCommunicateWithTheChildMachineInvokeOnMachine(t *testing.T) {
	sig := newSignal()
	subMachine := invoke1ParentToChildSubMachine()
	mainMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "one",
		Invoke:  []xs.InvokeConfig{{ID: "foo-child", Logic: subMachine}},
		States: xs.States{
			{
				Key:   "one",
				Entry: xs.Actions{xs.SendTo("foo-child", xs.Ev("NEXT"))},
				On:    map[string]xs.Transitions{"NEXT": {{Target: "two"}}},
			},
			{Key: "two", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(mainMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > parent to child > should communicate with the child machine (invoke on state)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L459
func TestInvoke_ParentToChild_ShouldCommunicateWithTheChildMachineInvokeOnState(t *testing.T) {
	sig := newSignal()
	subMachine := invoke1ParentToChildSubMachine()
	mainMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "one",
		States: xs.States{
			{
				Key:    "one",
				Invoke: []xs.InvokeConfig{{ID: "foo-child", Logic: subMachine}},
				Entry:  xs.Actions{xs.SendTo("foo-child", xs.Ev("NEXT"))},
				On:     map[string]xs.Transitions{"NEXT": {{Target: "two"}}},
			},
			{Key: "two", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(mainMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > parent to child > should transition correctly if child invocation causes it to directly go to final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L489
func TestInvoke_ParentToChild_ShouldTransitionCorrectlyIfChildInvocationGoesDirectlyToFinal(t *testing.T) {
	doneSubMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "one",
		States: xs.States{
			{Key: "one", On: map[string]xs.Transitions{"NEXT": {{Target: "two"}}}},
			{Key: "two", Type: xs.Final},
		},
	})

	mainMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "one",
		States: xs.States{
			{
				Key: "one",
				Invoke: []xs.InvokeConfig{{
					ID:     "foo-child",
					Logic:  doneSubMachine,
					OnDone: xs.Transitions{{Target: "two"}},
				}},
				Entry: xs.Actions{xs.SendTo("foo-child", xs.Ev("NEXT"))},
			},
			{Key: "two", On: map[string]xs.Transitions{"NEXT": {{Target: "three"}}}},
			{Key: "three", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(mainMachine).Start()

	assert.Equal(t, "two", actor.GetSnapshot().Value)
}

// JS: invoke > parent to child > should work with invocations defined in orthogonal state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L529
func TestInvoke_ParentToChild_ShouldWorkWithInvocationsDefinedInOrthogonalStateNodes(t *testing.T) {
	sig := newSignal()
	pongMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "pong",
		Initial: "active",
		States:  xs.States{{Key: "active", Type: xs.Final}},
		Output:  map[string]any{"secret": "pingpong"},
	})

	pingMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "ping",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "one",
				Initial: "active",
				States: xs.States{
					{
						Key: "active",
						Invoke: []xs.InvokeConfig{{
							ID:    "pong",
							Logic: pongMachine,
							OnDone: xs.Transitions{{
								Target: "success",
								Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
									return invoke1OutputField(a.Event, "secret") == "pingpong"
								}),
							}},
						}},
					},
					{Key: "success", Type: xs.Final},
				},
			},
		},
	})

	actor := xs.CreateActor(pingMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > parent to child > should not reinvoke root-level invocations on root non-reentering transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L577
func TestInvoke_ParentToChild_ShouldNotReinvokeRootLevelInvocationsOnRootNonReenteringTransitions(t *testing.T) {
	// https://github.com/statelyai/xstate/issues/2147

	var invokeCount, invokeDisposeCount, actionsCount, entryActionsCount atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
				invokeCount.Add(1)

				return func() {
					invokeDisposeCount.Add(1)
				}
			}),
		}},
		Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { entryActionsCount.Add(1) })},
		On: map[string]xs.Transitions{
			"UPDATE": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
				actionsCount.Add(1)
			})}}},
		},
	})

	service := xs.CreateActor(machine).Start()
	assert.Equal(t, int32(1), entryActionsCount.Load())
	assert.Equal(t, int32(1), invokeCount.Load())
	assert.Equal(t, int32(0), invokeDisposeCount.Load())
	assert.Equal(t, int32(0), actionsCount.Load())

	service.Send(xs.Ev("UPDATE"))
	assert.Equal(t, int32(1), entryActionsCount.Load())
	assert.Equal(t, int32(1), invokeCount.Load())
	assert.Equal(t, int32(0), invokeDisposeCount.Load())
	assert.Equal(t, int32(1), actionsCount.Load())

	service.Send(xs.Ev("UPDATE"))
	assert.Equal(t, int32(1), entryActionsCount.Load())
	assert.Equal(t, int32(1), invokeCount.Load())
	assert.Equal(t, int32(0), invokeDisposeCount.Load())
	assert.Equal(t, int32(2), actionsCount.Load())
}

// JS: invoke > parent to child > should stop a child actor when reaching a final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L624
func TestInvoke_ParentToChild_ShouldStopAChildActorWhenReachingAFinalState(t *testing.T) {
	var actorStopped atomic.Bool

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID: "machine",
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
				return func() { actorStopped.Store(true) }
			}),
		}},
		Initial: "running",
		States: xs.States{
			{Key: "running", On: map[string]xs.Transitions{"finished": {{Target: "complete"}}}},
			{Key: "complete", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("finished"))

	assert.True(t, actorStopped.Load())
}

// JS: invoke > parent to child > child should not invoke an actor when it transitions to an invoking state when it gets stopped by its parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L652
func TestInvoke_ParentToChild_ChildShouldNotInvokeActorWhenTransitioningToInvokingStateWhileStoppedByParent(t *testing.T) {
	sig := newSignal()
	var invokeCount atomic.Int32

	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "idle",
		States: xs.States{
			{
				Key: "idle",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						n := invokeCount.Add(1)

						if n > 1 {
							// prevent a potential infinite loop
							panic(errors.New("This should be impossible."))
						}

						// it's important for this test to send the event back when the parent is *not* currently processing an event
						// this ensures that the parent can process the received event immediately and can stop the child immediately
						time.AfterFunc(0, func() { a.SendBack(xs.Ev("STARTED")) })
						return nil
					}),
				}},
				On: map[string]xs.Transitions{"STARTED": {{Target: "active"}}},
			},
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						a.SendBack(xs.Ev("STOPPED"))
						return nil
					}),
				}},
				On: map[string]xs.Transitions{
					"STOPPED": {{
						Target: "idle",
						// SpecialTargets.Parent === '#_parent'
						Actions: xs.Actions{xs.ForwardTo("#_parent")},
					}},
				},
			},
		},
	})
	parent := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START": {{Target: "active"}}}},
			{
				Key:    "active",
				Invoke: []xs.InvokeConfig{{Logic: child}},
				On:     map[string]xs.Transitions{"STOPPED": {{Target: "done"}}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	service := xs.CreateActor(parent)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.Equal(t, int32(1), invokeCount.Load())
			sig.Resolve()
		},
	})
	service.Start()

	service.Send(xs.Ev("START"))
	sig.Wait(t)
}

// ---- describe(`with promises (${type})`) — bodies shared by both promise types ----

// JS: invoke > with promises (${type}) > should be invoked with a promise factory and resolve through onDone
func invoke1PromiseResolveThroughOnDone(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(resolve func(any), _ func(any)) {
							resolve(nil)
						})
					}),
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})
	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be invoked with a promise factory and reject with ErrorExecution
func invoke1PromiseRejectWithErrorExecution(t *testing.T, pt invoke1PromiseType) {
	type ctx struct {
		ID      int
		Succeed bool
	}

	sig := newSignal()
	// invokePromiseMachine from the describe body.
	invokePromiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "invokePromise",
		Initial: "pending",
		ContextFn: func(a xs.ContextArgs) ctx {
			c := ctx{ID: 42, Succeed: true}
			in, _ := a.Input.(map[string]any)
			if id, ok := in["id"].(int); ok {
				c.ID = id
			}
			if succeed, ok := in["succeed"].(bool); ok {
				c.Succeed = succeed
			}
			return c
		},
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, a xs.PromiseArgs) (any, error) {
						input := a.Input.(ctx)
						return pt.createPromise(c, func(resolve func(any), _ func(any)) {
							if input.Succeed {
								resolve(input.ID)
							} else {
								panic(fmt.Errorf("failed on purpose for: %d", input.ID))
							}
						})
					}),
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context }),
					OnDone: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return invoke1DoneOutput(a.Event) == a.Context.ID
						}),
					}},
					OnError: xs.Transitions{{Target: "failure"}},
				}},
			},
			{Key: "success", Type: xs.Final},
			{Key: "failure", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(invokePromiseMachine, xs.WithInput(map[string]any{"id": 31, "succeed": false}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be invoked with a promise factory and surface any unhandled errors
func invoke1PromiseSurfaceUnhandledErrors(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "invokePromise",
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(func(any), func(any)) {
							panic(errors.New("test"))
						})
					}),
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	service := xs.CreateActor(promiseMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			e, ok := err.(error)
			if assert.True(t, ok, "expected an error value, got %#v", err) {
				assert.Regexp(t, `test`, e.Error())
			}
			sig.Resolve()
		},
	})

	service.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be invoked with a promise factory and stop on unhandled onError target
func invoke1PromiseStopOnUnhandledOnErrorTarget(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	completeSpy := newSpy()

	promiseMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "invokePromise",
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(func(any), func(any)) {
							panic(errors.New("test"))
						})
					}),
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(promiseMachine)

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			e, ok := err.(error)
			if assert.True(t, ok, "expected an error value, got %#v", err) {
				assert.Equal(t, "test", e.Error())
			}
			assert.Equal(t, 0, completeSpy.Count())
			sig.Resolve()
		},
		Complete: func() { completeSpy.Call() },
	})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be invoked with a promise factory and resolve through onDone for compound state nodes
func invoke1PromiseFactoryResolveThroughOnDoneForCompound(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "promise",
		Initial: "parent",
		States: xs.States{
			{
				Key:     "parent",
				Initial: "pending",
				States: xs.States{
					{
						Key: "pending",
						Invoke: []xs.InvokeConfig{{
							Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
								return pt.createPromise(c, func(resolve func(any), _ func(any)) { resolve(nil) })
							}),
							OnDone: xs.Transitions{{Target: "success"}},
						}},
					},
					{Key: "success", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Target: "success"}},
			},
			{Key: "success", Type: xs.Final},
		},
	})
	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be invoked with a promise service and resolve through onDone for compound state nodes
func invoke1PromiseServiceResolveThroughOnDoneForCompound(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "promise",
		Initial: "parent",
		States: xs.States{
			{
				Key:     "parent",
				Initial: "pending",
				States: xs.States{
					{
						Key: "pending",
						Invoke: []xs.InvokeConfig{{
							Src:    "somePromise",
							OnDone: xs.Transitions{{Target: "success"}},
						}},
					},
					{Key: "success", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Target: "success"}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"somePromise": xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
				return pt.createPromise(c, func(resolve func(any), _ func(any)) { resolve(nil) })
			}),
		},
	})
	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should assign the resolved data when invoked with a promise factory
func invoke1PromiseFactoryAssignResolvedData(t *testing.T, pt invoke1PromiseType) {
	type ctx struct{ Count int }

	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Context: ctx{Count: 0},
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(resolve func(any), _ func(any)) {
							resolve(map[string]any{"count": 1})
						})
					}),
					OnDone: xs.Transitions{{
						Target: "success",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							c := a.Context
							c.Count, _ = invoke1OutputField(a.Event, "count").(int)
							return c
						})},
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			assert.Equal(t, 1, actor.GetSnapshot().Context.Count)
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should assign the resolved data when invoked with a promise service
func invoke1PromiseServiceAssignResolvedData(t *testing.T, pt invoke1PromiseType) {
	type ctx struct{ Count int }

	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Context: ctx{Count: 0},
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Src: "somePromise",
					OnDone: xs.Transitions{{
						Target: "success",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							c := a.Context
							c.Count, _ = invoke1OutputField(a.Event, "count").(int)
							return c
						})},
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"somePromise": xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
				return pt.createPromise(c, func(resolve func(any), _ func(any)) {
					resolve(map[string]any{"count": 1})
				})
			}),
		},
	})

	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			assert.Equal(t, 1, actor.GetSnapshot().Context.Count)
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should provide the resolved data when invoked with a promise factory
func invoke1PromiseFactoryProvideResolvedData(t *testing.T, pt invoke1PromiseType) {
	type ctx struct{ Count int }

	sig := newSignal()
	var count atomic.Int64

	promiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Context: ctx{Count: 0},
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(resolve func(any), _ func(any)) {
							resolve(map[string]any{"count": 1})
						})
					}),
					OnDone: xs.Transitions{{
						Target: "success",
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
							n, _ := invoke1OutputField(a.Event, "count").(int)
							count.Store(int64(n))
						})},
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			assert.Equal(t, int64(1), count.Load())
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should provide the resolved data when invoked with a promise service
func invoke1PromiseServiceProvideResolvedData(t *testing.T, pt invoke1PromiseType) {
	sig := newSignal()
	var count atomic.Int64

	promiseMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "promise",
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Src: "somePromise",
					OnDone: xs.Transitions{{
						Target: "success",
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
							n, _ := invoke1OutputField(a.Event, "count").(int)
							count.Store(int64(n))
						})},
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"somePromise": xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
				return pt.createPromise(c, func(resolve func(any), _ func(any)) {
					resolve(map[string]any{"count": 1})
				})
			}),
		},
	})

	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.Equal(t, int64(1), count.Load())
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be able to specify a Promise as a service
func invoke1PromiseSpecifyPromiseAsService(t *testing.T, pt invoke1PromiseType) {
	type ctx struct{ Foo bool }

	sig := newSignal()
	promiseActor := xs.FromPromise(func(c context.Context, a xs.PromiseArgs) (any, error) {
		input := a.Input.(map[string]any)
		return pt.createPromise(c, func(resolve func(any), reject func(any)) {
			foo, _ := input["foo"].(bool)
			ev, _ := input["event"].(xs.E)
			payload, _ := ev["payload"].(bool)
			if foo && payload {
				resolve(nil)
			} else {
				reject(nil)
			}
		})
	})

	promiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Initial: "pending",
		Context: ctx{Foo: true},
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{"BEGIN": {{Target: "first"}}}},
			{
				Key: "first",
				Invoke: []xs.InvokeConfig{{
					Src: "somePromise",
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return map[string]any{"foo": a.Context.Foo, "event": a.Event}
					}),
					OnDone: xs.Transitions{{Target: "last"}},
				}},
			},
			{Key: "last", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{"somePromise": promiseActor},
	})

	actor := xs.CreateActor(promiseMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	actor.Send(xs.E{"type": "BEGIN", "payload": true})
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should be able to reuse the same promise logic multiple times and create unique promise for each created actor
func invoke1PromiseReuseSameLogicUniquePromisePerActor(t *testing.T, pt invoke1PromiseType) {
	type ctx struct {
		Result1 *float64 // null when nil
		Result2 *float64 // null when nil
	}

	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Result1: nil, Result2: nil},
		Initial: "pending",
		States: xs.States{
			{
				Key:  "pending",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "state1",
						Initial: "active",
						States: xs.States{
							{
								Key: "active",
								Invoke: []xs.InvokeConfig{{
									Src: "getRandomNumber",
									OnDone: xs.Transitions{{
										Target: "success",
										Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
											c := a.Context
											if r, ok := invoke1OutputField(a.Event, "result").(float64); ok {
												c.Result1 = &r
											}
											return c
										})},
									}},
								}},
							},
							{Key: "success", Type: xs.Final},
						},
					},
					{
						Key:     "state2",
						Initial: "active",
						States: xs.States{
							{
								Key: "active",
								Invoke: []xs.InvokeConfig{{
									Src: "getRandomNumber",
									OnDone: xs.Transitions{{
										Target: "success",
										Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
											c := a.Context
											if r, ok := invoke1OutputField(a.Event, "result").(float64); ok {
												c.Result2 = &r
											}
											return c
										})},
									}},
								}},
							},
							{Key: "success", Type: xs.Final},
						},
					},
				},
				OnDone: xs.Transitions{{Target: "done"}},
			},
			{Key: "done", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			// it's important for this actor to be reused, this test shouldn't use a factory or anything like that
			"getRandomNumber": xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
				return pt.createPromise(c, func(resolve func(any), _ func(any)) {
					resolve(map[string]any{"result": rand.Float64()})
				})
			}),
		},
	})

	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			snapshot := service.GetSnapshot()
			// typeof result1 === 'number' / typeof result2 === 'number'
			if assert.NotNil(t, snapshot.Context.Result1) && assert.NotNil(t, snapshot.Context.Result2) {
				assert.NotEqual(t, *snapshot.Context.Result1, *snapshot.Context.Result2)
			}
			sig.Resolve()
		},
	})
	service.Start()
	sig.Wait(t)
}

// JS: invoke > with promises (${type}) > should not emit onSnapshot if stopped
func invoke1PromiseShouldNotEmitOnSnapshotIfStopped(t *testing.T, pt invoke1PromiseType) {
	// JS fails the test through the unhandled error raised when the '*'
	// action throws; the handler below makes that failure explicit.
	var mu sync.Mutex
	var unhandled []any

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (any, error) {
						return pt.createPromise(c, func(res func(any), _ func(any)) {
							time.AfterFunc(ms(5), func() { res(42) })
						})
					}),
					OnSnapshot: xs.Transitions{{}},
				}},
				On: map[string]xs.Transitions{"deactivate": {{Target: "inactive"}}},
			},
			{
				Key: "inactive",
				On: map[string]xs.Transitions{
					"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
						if _, ok := a.Event.(xs.SnapshotEvent); ok {
							panic(fmt.Errorf("Received unexpected event: %s", a.Event.EventType()))
						}
					})}}},
				},
			},
		},
	})

	actor := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(func(err any) {
		mu.Lock()
		unhandled = append(unhandled, err)
		mu.Unlock()
	})).Start()
	actor.Send(xs.Ev("deactivate"))

	sleep(10)

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, unhandled)
	assert.NotEqual(t, xs.StatusError, actor.GetSnapshot().Status)
}

// ---- describe(`with promises (Promise)`) ----

// JS: invoke > with promises (Promise) > should be invoked with a promise factory and resolve through onDone
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L803
func TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndResolveThroughOnDone(t *testing.T) {
	invoke1PromiseResolveThroughOnDone(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be invoked with a promise factory and reject with ErrorExecution
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L833
func TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndRejectWithErrorExecution(t *testing.T) {
	invoke1PromiseRejectWithErrorExecution(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be invoked with a promise factory and surface any unhandled errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L843
func TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndSurfaceAnyUnhandledErrors(t *testing.T) {
	invoke1PromiseSurfaceUnhandledErrors(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be invoked with a promise factory and stop on unhandled onError target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L877
func TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndStopOnUnhandledOnErrorTarget(t *testing.T) {
	invoke1PromiseStopOnUnhandledOnErrorTarget(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be invoked with a promise factory and resolve through onDone for compound state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L916
func TestInvoke_WithPromisesPromise_PromiseFactoryResolveThroughOnDoneForCompoundStateNodes(t *testing.T) {
	invoke1PromiseFactoryResolveThroughOnDoneForCompound(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be invoked with a promise service and resolve through onDone for compound state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L950
func TestInvoke_WithPromisesPromise_PromiseServiceResolveThroughOnDoneForCompoundStateNodes(t *testing.T) {
	invoke1PromiseServiceResolveThroughOnDoneForCompound(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should assign the resolved data when invoked with a promise factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L990
func TestInvoke_WithPromisesPromise_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseFactory(t *testing.T) {
	invoke1PromiseFactoryAssignResolvedData(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should assign the resolved data when invoked with a promise service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1028
func TestInvoke_WithPromisesPromise_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseService(t *testing.T) {
	invoke1PromiseServiceAssignResolvedData(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should provide the resolved data when invoked with a promise factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1073
func TestInvoke_WithPromisesPromise_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseFactory(t *testing.T) {
	invoke1PromiseFactoryProvideResolvedData(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should provide the resolved data when invoked with a promise service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1112
func TestInvoke_WithPromisesPromise_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseService(t *testing.T) {
	invoke1PromiseServiceProvideResolvedData(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be able to specify a Promise as a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1157
func TestInvoke_WithPromisesPromise_ShouldBeAbleToSpecifyAPromiseAsAService(t *testing.T) {
	invoke1PromiseSpecifyPromiseAsService(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should be able to reuse the same promise logic multiple times and create unique promise for each created actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1225
func TestInvoke_WithPromisesPromise_ShouldReuseSamePromiseLogicAndCreateUniquePromisePerActor(t *testing.T) {
	invoke1PromiseReuseSameLogicUniquePromisePerActor(t, invoke1PromiseTypes["Promise"])
}

// JS: invoke > with promises (Promise) > should not emit onSnapshot if stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1321
func TestInvoke_WithPromisesPromise_ShouldNotEmitOnSnapshotIfStopped(t *testing.T) {
	invoke1PromiseShouldNotEmitOnSnapshotIfStopped(t, invoke1PromiseTypes["Promise"])
}

// ---- describe(`with promises (PromiseLike)`) ----

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise factory and resolve through onDone
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L803
func TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndResolveThroughOnDone(t *testing.T) {
	invoke1PromiseResolveThroughOnDone(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise factory and reject with ErrorExecution
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L833
func TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndRejectWithErrorExecution(t *testing.T) {
	invoke1PromiseRejectWithErrorExecution(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise factory and surface any unhandled errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L843
func TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndSurfaceAnyUnhandledErrors(t *testing.T) {
	invoke1PromiseSurfaceUnhandledErrors(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise factory and stop on unhandled onError target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L877
func TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndStopOnUnhandledOnErrorTarget(t *testing.T) {
	invoke1PromiseStopOnUnhandledOnErrorTarget(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise factory and resolve through onDone for compound state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L916
func TestInvoke_WithPromisesPromiseLike_PromiseFactoryResolveThroughOnDoneForCompoundStateNodes(t *testing.T) {
	invoke1PromiseFactoryResolveThroughOnDoneForCompound(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be invoked with a promise service and resolve through onDone for compound state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L950
func TestInvoke_WithPromisesPromiseLike_PromiseServiceResolveThroughOnDoneForCompoundStateNodes(t *testing.T) {
	invoke1PromiseServiceResolveThroughOnDoneForCompound(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should assign the resolved data when invoked with a promise factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L990
func TestInvoke_WithPromisesPromiseLike_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseFactory(t *testing.T) {
	invoke1PromiseFactoryAssignResolvedData(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should assign the resolved data when invoked with a promise service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1028
func TestInvoke_WithPromisesPromiseLike_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseService(t *testing.T) {
	invoke1PromiseServiceAssignResolvedData(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should provide the resolved data when invoked with a promise factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1073
func TestInvoke_WithPromisesPromiseLike_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseFactory(t *testing.T) {
	invoke1PromiseFactoryProvideResolvedData(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should provide the resolved data when invoked with a promise service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1112
func TestInvoke_WithPromisesPromiseLike_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseService(t *testing.T) {
	invoke1PromiseServiceProvideResolvedData(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be able to specify a Promise as a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1157
func TestInvoke_WithPromisesPromiseLike_ShouldBeAbleToSpecifyAPromiseAsAService(t *testing.T) {
	invoke1PromiseSpecifyPromiseAsService(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should be able to reuse the same promise logic multiple times and create unique promise for each created actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1225
func TestInvoke_WithPromisesPromiseLike_ShouldReuseSamePromiseLogicAndCreateUniquePromisePerActor(t *testing.T) {
	invoke1PromiseReuseSameLogicUniquePromisePerActor(t, invoke1PromiseTypes["PromiseLike"])
}

// JS: invoke > with promises (PromiseLike) > should not emit onSnapshot if stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1321
func TestInvoke_WithPromisesPromiseLike_ShouldNotEmitOnSnapshotIfStopped(t *testing.T) {
	invoke1PromiseShouldNotEmitOnSnapshotIfStopped(t, invoke1PromiseTypes["PromiseLike"])
}

// invoke2EventField returns event[key] of a dynamic xs.E event (nil otherwise).
func invoke2EventField(e xs.Event, key string) any {
	m, ok := e.(xs.E)
	if !ok {
		return nil
	}
	return m[key]
}

// invoke2ErrorMessage returns `event.error.message` of an
// "xstate.error.actor.*" event; ok is false when event.error is not an error
// (JS `event.error instanceof Error`).
func invoke2ErrorMessage(e xs.Event) (msg string, ok bool) {
	ev, isErrEv := e.(xs.ErrorActorEvent)
	if !isErrEv {
		return "", false
	}
	err, isErr := ev.Error.(error)
	if !isErr {
		return "", false
	}
	return err.Error(), true
}

// invoke2SnapshotOf returns `event.snapshot` of an "xstate.snapshot.*" event.
func invoke2SnapshotOf(e xs.Event) xs.Snapshot {
	ev, ok := e.(xs.SnapshotEvent)
	if !ok {
		return nil
	}
	return ev.Snapshot
}

// invoke2ObservableCtx returns `event.snapshot.context` of an
// "xstate.snapshot.*" event emitted by an observable child of ints.
func invoke2ObservableCtx(e xs.Event) (int, bool) {
	s, ok := invoke2SnapshotOf(e).(*xs.ObservableSnapshot[int])
	if !ok {
		return 0, false
	}
	return s.Context, true
}

// invoke2IntPtr mirrors a `number` that can be `undefined` in JS contexts.
func invoke2IntPtr(v int) *int { return &v }

// invoke2MapOrThrow mirrors rxjs `pipe(map(fn))` where fn throws
// `new Error(msg)` for the value throwAt: the error is delivered to the
// observer's error callback and the source subscription is torn down.
func invoke2MapOrThrow[U any](src xs.Subscribable[int], throwAt int, msg string, fn func(int) U) xs.Subscribable[U] {
	return &observable[U]{subscribe: func(o xs.Observer[U]) func() {
		var mu sync.Mutex
		var sub xs.Subscription
		failed := false
		sub = src.Subscribe(xs.Observer[int]{
			Next: func(v int) {
				if v == throwAt {
					mu.Lock()
					failed = true
					s := sub
					mu.Unlock()
					o.Error(errors.New(msg))
					if s != nil {
						s.Unsubscribe()
					}
					return
				}
				o.Next(fn(v))
			},
			Error:    o.Error,
			Complete: o.Complete,
		})
		mu.Lock()
		s := sub
		f := failed
		mu.Unlock()
		if f {
			s.Unsubscribe()
		}
		return func() { s.Unsubscribe() }
	}}
}

// invoke2Ticker mirrors `setInterval(fn, period)`; the returned func mirrors
// `clearInterval`.
func invoke2Ticker(period int, fn func()) func() {
	stop := make(chan struct{})
	go func() {
		tk := time.NewTicker(ms(period))
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				fn()
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(stop) }) }
}

// ---- with callbacks ----

// JS: invoke > with callbacks > should be able to specify a callback as a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1367
func TestInvoke_WithCallbacks_ShouldBeAbleToSpecifyACallbackAsAService(t *testing.T) {
	type ctx struct{ Foo bool }
	type callbackInput struct {
		Foo   bool
		Event xs.Event
	}

	sig := newSignal()

	someCallback := xs.FromCallback(func(a xs.CallbackArgs) func() {
		input := a.Input.(callbackInput)
		if input.Foo && input.Event.EventType() == "BEGIN" {
			a.SendBack(xs.E{"type": "CALLBACK", "data": 40})
			a.SendBack(xs.E{"type": "CALLBACK", "data": 41})
			a.SendBack(xs.E{"type": "CALLBACK", "data": 42})
		}
		return nil
	})

	callbackMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "callback",
		Initial: "pending",
		Context: ctx{Foo: true},
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{"BEGIN": {{Target: "first"}}}},
			{
				Key: "first",
				Invoke: []xs.InvokeConfig{{
					Src: "someCallback",
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return callbackInput{Foo: a.Context.Foo, Event: a.Event}
					}),
				}},
				On: map[string]xs.Transitions{
					"CALLBACK": {{
						Target: "last",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return invoke2EventField(a.Event, "data") == 42
						}),
					}},
				},
			},
			{Key: "last", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"someCallback": someCallback}})

	actor := xs.CreateActor(callbackMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	actor.Send(xs.E{"type": "BEGIN", "payload": true})
	sig.Wait(t)
}

// JS: invoke > with callbacks > should transition correctly if callback function sends an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1461
func TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfCallbackFunctionSendsAnEvent(t *testing.T) {
	type ctx struct{ Foo bool }

	callbackMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "callback",
		Initial: "pending",
		Context: ctx{Foo: true},
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{"BEGIN": {{Target: "first"}}}},
			{
				Key:    "first",
				Invoke: []xs.InvokeConfig{{Src: "someCallback"}},
				On:     map[string]xs.Transitions{"CALLBACK": {{Target: "intermediate"}}},
			},
			{Key: "intermediate", On: map[string]xs.Transitions{"NEXT": {{Target: "last"}}}},
			{Key: "last", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"someCallback": xs.FromCallback(func(a xs.CallbackArgs) func() {
			a.SendBack(xs.Ev("CALLBACK"))
			return nil
		}),
	}})

	expectedStateValues := []xs.StateValue{"pending", "first", "intermediate"}
	var mu sync.Mutex
	var stateValues []xs.StateValue
	actor := xs.CreateActor(callbackMachine)
	actor.SubscribeNext(func(current *xs.MachineSnapshot[ctx]) {
		mu.Lock()
		defer mu.Unlock()
		stateValues = append(stateValues, current.Value)
	})
	actor.Start().Send(xs.Ev("BEGIN"))

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(stateValues), len(expectedStateValues))
	for i := range expectedStateValues {
		assert.Equal(t, expectedStateValues[i], stateValues[i])
	}
}

// JS: invoke > with callbacks > should transition correctly if callback function invoked from start and sends an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1504
func TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfCallbackInvokedFromStartAndSendsAnEvent(t *testing.T) {
	type ctx struct{ Foo bool }

	callbackMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "callback",
		Initial: "idle",
		Context: ctx{Foo: true},
		States: xs.States{
			{
				Key:    "idle",
				Invoke: []xs.InvokeConfig{{Src: "someCallback"}},
				On:     map[string]xs.Transitions{"CALLBACK": {{Target: "intermediate"}}},
			},
			{Key: "intermediate", On: map[string]xs.Transitions{"NEXT": {{Target: "last"}}}},
			{Key: "last", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"someCallback": xs.FromCallback(func(a xs.CallbackArgs) func() {
			a.SendBack(xs.Ev("CALLBACK"))
			return nil
		}),
	}})

	expectedStateValues := []xs.StateValue{"idle", "intermediate"}
	var mu sync.Mutex
	var stateValues []xs.StateValue
	actor := xs.CreateActor(callbackMachine)
	actor.SubscribeNext(func(current *xs.MachineSnapshot[ctx]) {
		mu.Lock()
		defer mu.Unlock()
		stateValues = append(stateValues, current.Value)
	})
	actor.Start().Send(xs.Ev("BEGIN"))

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(stateValues), len(expectedStateValues))
	for i := range expectedStateValues {
		assert.Equal(t, expectedStateValues[i], stateValues[i])
	}
}

// JS: invoke > with callbacks > should transition correctly if transient transition happens before current state invokes callback function and sends an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1545
func TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfTransientTransitionHappensBeforeInvokingCallback(t *testing.T) {
	type ctx struct{ Foo bool }

	callbackMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "callback",
		Initial: "pending",
		Context: ctx{Foo: true},
		States: xs.States{
			{Key: "pending", On: map[string]xs.Transitions{"BEGIN": {{Target: "first"}}}},
			{Key: "first", Always: xs.Transitions{{Target: "second"}}},
			{
				Key:    "second",
				Invoke: []xs.InvokeConfig{{Src: "someCallback"}},
				On:     map[string]xs.Transitions{"CALLBACK": {{Target: "third"}}},
			},
			{Key: "third", On: map[string]xs.Transitions{"NEXT": {{Target: "last"}}}},
			{Key: "last", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"someCallback": xs.FromCallback(func(a xs.CallbackArgs) func() {
			a.SendBack(xs.Ev("CALLBACK"))
			return nil
		}),
	}})

	expectedStateValues := []xs.StateValue{"pending", "second", "third"}
	var mu sync.Mutex
	var stateValues []xs.StateValue
	actor := xs.CreateActor(callbackMachine)
	actor.SubscribeNext(func(current *xs.MachineSnapshot[ctx]) {
		mu.Lock()
		defer mu.Unlock()
		stateValues = append(stateValues, current.Value)
	})
	actor.Start().Send(xs.Ev("BEGIN"))

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(stateValues), len(expectedStateValues))
	for i := range expectedStateValues {
		assert.Equal(t, expectedStateValues[i], stateValues[i])
	}
}

// JS: invoke > with callbacks > should treat a callback source as an event stream
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1594
func TestInvoke_WithCallbacks_ShouldTreatACallbackSourceAsAnEventStream(t *testing.T) {
	type ctx struct{ Count int }

	sig := newSignal()
	intervalMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "interval",
		Initial: "counting",
		Context: ctx{Count: 0},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					ID: "intervalService",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						clearInterval := invoke2Ticker(10, func() { a.SendBack(xs.Ev("INC")) })
						return clearInterval
					}),
				}},
				Always: xs.Transitions{{
					Target: "finished",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count == 3 }),
				}},
				On: map[string]xs.Transitions{
					"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: a.Context.Count + 1}
					})}}},
				},
			},
			{Key: "finished", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(intervalMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with callbacks > should dispose of the callback (if disposal function provided)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1636
func TestInvoke_WithCallbacks_ShouldDisposeOfTheCallbackIfDisposalFunctionProvided(t *testing.T) {
	s := newSpy()
	intervalMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "interval",
		Initial: "counting",
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					ID:    "intervalService",
					Logic: xs.FromCallback(func(xs.CallbackArgs) func() { return func() { s.Call() } }),
				}},
				On: map[string]xs.Transitions{"NEXT": {{Target: "idle"}}},
			},
			{Key: "idle"},
		},
	})
	actorRef := xs.CreateActor(intervalMachine).Start()

	actorRef.Send(xs.Ev("NEXT"))

	// expect(spy).toHaveBeenCalled()
	assert.GreaterOrEqual(t, s.Count(), 1)
}

// JS: invoke > with callbacks > callback should be able to receive messages from parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1661
func TestInvoke_WithCallbacks_CallbackShouldBeAbleToReceiveMessagesFromParent(t *testing.T) {
	sig := newSignal()
	pingPongMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "ping-pong",
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID: "child",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						a.Receive(func(e xs.Event) {
							if e.EventType() == "PING" {
								a.SendBack(xs.Ev("PONG"))
							}
						})
						return nil
					}),
				}},
				Entry: xs.Actions{xs.SendTo("child", xs.Ev("PING"))},
				On:    map[string]xs.Transitions{"PONG": {{Target: "done"}}},
			},
			{Key: "done", Type: xs.Final},
		},
	})
	actor := xs.CreateActor(pingPongMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with callbacks > should call onError upon error (sync)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1694
func TestInvoke_WithCallbacks_ShouldCallOnErrorUponErrorSync(t *testing.T) {
	sig := newSignal()
	errorMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "error",
		Initial: "safe",
		States: xs.States{
			{
				Key: "safe",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(xs.CallbackArgs) func() { panic(errors.New("test")) }),
					OnError: xs.Transitions{{
						Target: "failed",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							msg, ok := invoke2ErrorMessage(a.Event)
							return ok && msg == "test"
						}),
					}},
				}},
			},
			{Key: "failed", Type: xs.Final},
		},
	})
	actor := xs.CreateActor(errorMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with callbacks > should transition correctly upon error (sync)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1727
func TestInvoke_WithCallbacks_ShouldTransitionCorrectlyUponErrorSync(t *testing.T) {
	errorMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "error",
		Initial: "safe",
		States: xs.States{
			{
				Key: "safe",
				Invoke: []xs.InvokeConfig{{
					Logic:   xs.FromCallback(func(xs.CallbackArgs) func() { panic(errors.New("test")) }),
					OnError: xs.Transitions{{Target: "failed"}},
				}},
			},
			{Key: "failed", On: map[string]xs.Transitions{"RETRY": {{Target: "safe"}}}},
		},
	})

	expectedStateValue := "failed"
	service := xs.CreateActor(errorMachine).Start()
	assert.Equal(t, expectedStateValue, service.GetSnapshot().Value)
}

// JS: invoke > with callbacks > should call onError only on the state which has invoked failed service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1751
func TestInvoke_WithCallbacks_ShouldCallOnErrorOnlyOnTheStateWhichHasInvokedFailedService(t *testing.T) {
	errorMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"FETCH": {{Target: "fetch"}}}},
			{
				Key:  "fetch",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "first",
						Initial: "waiting",
						States: xs.States{
							{
								Key: "waiting",
								Invoke: []xs.InvokeConfig{{
									Logic:   xs.FromCallback(func(xs.CallbackArgs) func() { panic(errors.New("test")) }),
									OnError: xs.Transitions{{Target: "failed"}},
								}},
							},
							{Key: "failed"},
						},
					},
					{
						Key:     "second",
						Initial: "waiting",
						States: xs.States{
							{
								Key: "waiting",
								Invoke: []xs.InvokeConfig{{
									Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
										// empty
										return func() {}
									}),
									OnError: xs.Transitions{{Target: "failed"}},
								}},
							},
							{Key: "failed"},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(errorMachine).Start()
	actorRef.Send(xs.Ev("FETCH"))

	assert.Equal(t, map[string]any{
		"fetch": map[string]any{"first": "failed", "second": "waiting"},
	}, actorRef.GetSnapshot().Value)
}

// JS: invoke > with callbacks > should be able to be stringified
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1809
func TestInvoke_WithCallbacks_ShouldBeAbleToBeStringified(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"GO_TO_WAITING": {{Target: "waiting"}}}},
			{
				Key: "waiting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(xs.CallbackArgs) func() { return nil }),
				}},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("GO_TO_WAITING"))
	waitingState := actorRef.GetSnapshot()

	// JSON.stringify(waitingState) uses snapshot.toJSON().
	assert.NotPanics(t, func() {
		_, err := json.Marshal(waitingState.ToJSON())
		assert.NoError(t, err)
	})
}

// JS: invoke > with callbacks > should result in an error notification if callback actor throws when it starts and the error stays unhandled by the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1834
func TestInvoke_WithCallbacks_ShouldResultInErrorNotificationIfCallbackThrowsOnStartUnhandled(t *testing.T) {
	errorMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "safe",
		States: xs.States{
			{
				Key: "safe",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(xs.CallbackArgs) func() { panic(errors.New("test")) }),
				}},
			},
			{Key: "failed", Type: xs.Final},
		},
	})
	s := newSpy()

	actorRef := xs.CreateActor(errorMachine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { s.Call(err) },
	})
	actorRef.Start()

	// toMatchInlineSnapshot: [[ [Error: test] ]]
	calls := s.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %#v", calls[0][0])
	assert.EqualError(t, err, "test")
}

// JS: invoke > with callbacks > should work with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1866
func TestInvoke_WithCallbacks_ShouldWorkWithInput(t *testing.T) {
	type ctx struct{ Foo string }

	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "start",
		Context: ctx{Foo: "bar"},
		States: xs.States{
			{
				Key: "start",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						assert.Equal(t, ctx{Foo: "bar"}, a.Input)
						sig.Resolve()
						return nil
					}),
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context }),
				}},
			},
		},
	})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// JS: invoke > with callbacks > sub invoke race condition ends on the completed state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1891
func TestInvoke_WithCallbacks_SubInvokeRaceConditionEndsOnTheCompletedState(t *testing.T) {
	anotherChildMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"STOP": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final},
		},
	})

	anotherParentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "begin",
		States: xs.States{
			{
				Key: "begin",
				Invoke: []xs.InvokeConfig{{
					Logic:  anotherChildMachine,
					ID:     "invoked.child",
					OnDone: xs.Transitions{{Target: "completed"}},
				}},
				On: map[string]xs.Transitions{
					"STOPCHILD": {{Actions: xs.Actions{xs.SendTo("invoked.child", xs.Ev("STOP"))}}},
				},
			},
			{Key: "completed", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(anotherParentMachine).Start()
	actorRef.Send(xs.Ev("STOPCHILD"))

	assert.Equal(t, "completed", actorRef.GetSnapshot().Value)
}

// ---- with observables ----

// JS: invoke > with observables > should work with an infinite observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1935
func TestInvoke_WithObservables_ShouldWorkWithAnInfiniteObservable(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "infiniteObs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) }),
					OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						v, ok := invoke2ObservableCtx(a.Event)
						if !ok {
							return ctx{Count: nil}
						}
						return ctx{Count: invoke2IntPtr(v)}
					})}}},
				}},
				Always: xs.Transitions{{
					Target: "counted",
					Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
						return a.Context.Count != nil && *a.Context.Count == 5
					}),
				}},
			},
			{Key: "counted", Type: xs.Final},
		},
	})

	service := xs.CreateActor(obsMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	service.Start()
	sig.Wait(t)
}

// JS: invoke > with observables > should work with a finite observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L1977
func TestInvoke_WithObservables_ShouldWorkWithAFiniteObservable(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "obs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
						return rxTake(rxInterval(10), 5)
					}),
					OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						v, ok := invoke2ObservableCtx(a.Event)
						if !ok {
							return ctx{Count: nil}
						}
						return ctx{Count: invoke2IntPtr(v)}
					})}}},
					OnDone: xs.Transitions{{
						Target: "counted",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return a.Context.Count != nil && *a.Context.Count == 4
						}),
					}},
				}},
			},
			{Key: "counted", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(obsMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with observables > should receive an emitted error
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2024
func TestInvoke_WithObservables_ShouldReceiveAnEmittedError(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "obs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
						return invoke2MapOrThrow(rxInterval(10), 5, "some error", func(v int) int { return v })
					}),
					OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						v, ok := invoke2ObservableCtx(a.Event)
						if !ok {
							return ctx{Count: nil}
						}
						return ctx{Count: invoke2IntPtr(v)}
					})}}},
					OnError: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							msg, _ := invoke2ErrorMessage(a.Event)
							assert.Equal(t, "some error", msg)
							return a.Context.Count != nil && *a.Context.Count == 4 && msg == "some error"
						}),
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(obsMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with observables > should work with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2085
func TestInvoke_WithObservables_ShouldWorkWithInput(t *testing.T) {
	type ctx struct{ Received any }

	sig := newSignal()
	childLogic := xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[int] {
		return rxOf(a.Input.(int))
	})

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Received: nil},
		Invoke: []xs.InvokeConfig{{
			Src:   "childLogic",
			Input: 42,
			OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
				s, ok := invoke2SnapshotOf(a.Event).(*xs.ObservableSnapshot[int])
				if ok && s.Status == xs.StatusActive && s.Context == 42 {
					sig.Resolve()
				}
			})}}},
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"childLogic": childLogic}})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// ---- with event observables ----

// JS: invoke > with event observables > should work with an infinite event observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2128
func TestInvoke_WithEventObservables_ShouldWorkWithAnInfiniteEventObservable(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "obs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromEventObservable(func(xs.ObservableArgs) xs.Subscribable[xs.Event] {
						return rxMap(rxInterval(10), func(v int) xs.Event { return xs.E{"type": "COUNT", "value": v} })
					}),
				}},
				On: map[string]xs.Transitions{
					"COUNT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: invoke2IntPtr(invoke2EventField(a.Event, "value").(int))}
					})}}},
				},
				Always: xs.Transitions{{
					Target: "counted",
					Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
						return a.Context.Count != nil && *a.Context.Count == 5
					}),
				}},
			},
			{Key: "counted", Type: xs.Final},
		},
	})

	service := xs.CreateActor(obsMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	service.Start()
	sig.Wait(t)
}

// JS: invoke > with event observables > should work with a finite event observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2172
func TestInvoke_WithEventObservables_ShouldWorkWithAFiniteEventObservable(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "obs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromEventObservable(func(xs.ObservableArgs) xs.Subscribable[xs.Event] {
						return rxMap(rxTake(rxInterval(10), 5), func(v int) xs.Event {
							return xs.E{"type": "COUNT", "value": v}
						})
					}),
					OnDone: xs.Transitions{{
						Target: "counted",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return a.Context.Count != nil && *a.Context.Count == 4
						}),
					}},
				}},
				On: map[string]xs.Transitions{
					"COUNT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: invoke2IntPtr(invoke2EventField(a.Event, "value").(int))}
					})}}},
				},
			},
			{Key: "counted", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(obsMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with event observables > should receive an emitted error
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2226
func TestInvoke_WithEventObservables_ShouldReceiveAnEmittedError(t *testing.T) {
	type ctx struct{ Count *int }

	sig := newSignal()
	obsMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "obs",
		Initial: "counting",
		Context: ctx{Count: nil},
		States: xs.States{
			{
				Key: "counting",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromEventObservable(func(xs.ObservableArgs) xs.Subscribable[xs.Event] {
						return invoke2MapOrThrow(rxInterval(10), 5, "some error", func(v int) xs.Event {
							return xs.E{"type": "COUNT", "value": v}
						})
					}),
					OnError: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							msg, _ := invoke2ErrorMessage(a.Event)
							assert.Equal(t, "some error", msg)
							return a.Context.Count != nil && *a.Context.Count == 4 && msg == "some error"
						}),
					}},
				}},
				On: map[string]xs.Transitions{
					"COUNT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: invoke2IntPtr(invoke2EventField(a.Event, "value").(int))}
					})}}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(obsMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with event observables > should work with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2287
func TestInvoke_WithEventObservables_ShouldWorkWithInput(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromEventObservable(func(a xs.ObservableArgs) xs.Subscribable[xs.Event] {
				return rxOf[xs.Event](xs.E{"type": "obs.event", "value": a.Input})
			}),
			Input: 42,
		}},
		On: map[string]xs.Transitions{
			"obs.event": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				assert.Equal(t, 42, invoke2EventField(a.Event, "value"))
				sig.Resolve()
			})}}},
		},
	})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// ---- with logic ----

// JS: invoke > with logic > should work with actor logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2315
func TestInvoke_WithLogic_ShouldWorkWithActorLogic(t *testing.T) {
	sig := newSignal()
	countLogic := &xs.Logic[*xs.BasicSnapshot[int]]{
		Transition: func(state *xs.BasicSnapshot[int], event xs.Event, _ *xs.ActorScope) *xs.BasicSnapshot[int] {
			switch event.EventType() {
			case "INC":
				next := *state
				next.Context = state.Context + 1
				return &next
			case "DEC":
				next := *state
				next.Context = state.Context - 1
				return &next
			}
			return state
		},
		GetInitialSnapshot: func(*xs.ActorScope, any) *xs.BasicSnapshot[int] {
			return &xs.BasicSnapshot[int]{Status: xs.StatusActive, Output: nil, Error: nil, Context: 0}
		},
		GetPersistedSnapshot: func(s *xs.BasicSnapshot[int]) any { return s },
	}

	countMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{ID: "count", Logic: countLogic}},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.ForwardTo("count")}}},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.SubscribeNext(func(state *xs.MachineSnapshot[any]) {
		child := state.Children["count"]
		if child == nil {
			return
		}
		if s, ok := child.AnySnapshot().(*xs.BasicSnapshot[int]); ok && s.Context == 2 {
			sig.Resolve()
		}
	})
	countService.Start()

	countService.Send(xs.Ev("INC"))
	countService.Send(xs.Ev("INC"))
	sig.Wait(t)
}

// JS: invoke > with logic > logic should have reference to the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2369
func TestInvoke_WithLogic_LogicShouldHaveReferenceToTheParent(t *testing.T) {
	sig := newSignal()
	pongLogic := &xs.Logic[*xs.BasicSnapshot[any]]{
		Transition: func(state *xs.BasicSnapshot[any], event xs.Event, scope *xs.ActorScope) *xs.BasicSnapshot[any] {
			if event.EventType() == "PING" {
				if p := scope.Self.Parent(); p != nil {
					p.Send(xs.Ev("PONG"))
				}
			}
			return state
		},
		GetInitialSnapshot: func(*xs.ActorScope, any) *xs.BasicSnapshot[any] {
			return &xs.BasicSnapshot[any]{Status: xs.StatusActive, Output: nil, Error: nil}
		},
		GetPersistedSnapshot: func(s *xs.BasicSnapshot[any]) any { return s },
	}

	pingMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{
				Key:    "waiting",
				Entry:  xs.Actions{xs.SendTo("ponger", xs.Ev("PING"))},
				Invoke: []xs.InvokeConfig{{ID: "ponger", Logic: pongLogic}},
				On:     map[string]xs.Transitions{"PONG": {{Target: "success"}}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	pingService := xs.CreateActor(pingMachine)
	pingService.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: sig.Resolve})
	pingService.Start()
	sig.Wait(t)
}

// ---- with transition functions ----

// JS: invoke > with transition functions > should work with a transition function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2418
func TestInvoke_WithTransitionFunctions_ShouldWorkWithATransitionFunction(t *testing.T) {
	sig := newSignal()
	countReducer := func(count int, event xs.Event, _ *xs.ActorScope) int {
		switch event.EventType() {
		case "INC":
			return count + 1
		case "DEC":
			return count - 1
		}
		return count
	}

	countMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID:    "count",
			Logic: xs.FromTransition(countReducer, func(xs.TransitionInitArgs) int { return 0 }),
		}},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.ForwardTo("count")}}},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.SubscribeNext(func(state *xs.MachineSnapshot[any]) {
		child := state.Children["count"]
		if child == nil {
			return
		}
		if s, ok := child.AnySnapshot().(*xs.TransitionSnapshot[int]); ok && s.Context == 2 {
			sig.Resolve()
		}
	})
	countService.Start()

	countService.Send(xs.Ev("INC"))
	countService.Send(xs.Ev("INC"))
	sig.Wait(t)
}

// JS: invoke > with transition functions > should schedule events in a FIFO queue
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2457
func TestInvoke_WithTransitionFunctions_ShouldScheduleEventsInAFIFOQueue(t *testing.T) {
	sig := newSignal()
	countReducer := func(count int, event xs.Event, scope *xs.ActorScope) int {
		if event.EventType() == "INC" {
			scope.Self.Send(xs.Ev("DOUBLE"))
			return count + 1
		}
		if event.EventType() == "DOUBLE" {
			return count * 2
		}
		return count
	}

	countMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID:    "count",
			Logic: xs.FromTransition(countReducer, func(xs.TransitionInitArgs) int { return 0 }),
		}},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.ForwardTo("count")}}},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.SubscribeNext(func(state *xs.MachineSnapshot[any]) {
		child := state.Children["count"]
		if child == nil {
			return
		}
		if s, ok := child.AnySnapshot().(*xs.TransitionSnapshot[int]); ok && s.Context == 2 {
			sig.Resolve()
		}
	})
	countService.Start()

	countService.Send(xs.Ev("INC"))
	sig.Wait(t)
}

// JS: invoke > with transition functions > should emit onSnapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2501
func TestInvoke_WithTransitionFunctions_ShouldEmitOnSnapshot(t *testing.T) {
	sig := newSignal()
	doublerLogic := xs.FromTransition(
		func(_ int, event xs.Event, _ *xs.ActorScope) int {
			v, _ := invoke2EventField(event, "value").(int)
			return v * 2
		},
		func(xs.TransitionInitArgs) int { return 0 },
	)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID:  "doubler",
			Src: "doublerLogic",
			OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				s, ok := invoke2SnapshotOf(a.Event).(*xs.TransitionSnapshot[int])
				if ok && s.Context == 42 {
					sig.Resolve()
				}
			})}}},
		}},
		Entry: xs.Actions{xs.SendTo("doubler", xs.E{"type": "update", "value": 21}, xs.SendOptions{Delay: 10 * time.Millisecond})},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"doublerLogic": doublerLogic}})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// ---- with machines ----

// JS: invoke > with machines > should create invocations from machines in nested states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2582
func TestInvoke_WithMachines_ShouldCreateInvocationsFromMachinesInNestedStates(t *testing.T) {
	// describe-level fixtures `pongMachine` / `pingMachine` (invoke.test.ts:2538-2580).
	pongMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "pong",
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				On: map[string]xs.Transitions{
					// Sends 'PONG' event to parent machine
					"PING": {{Actions: xs.Actions{xs.SendParent(xs.Ev("PONG"))}}},
				},
			},
		},
	})

	// Parent machine
	pingMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "ping",
		Initial: "innerMachine",
		States: xs.States{
			{
				Key:     "innerMachine",
				Initial: "active",
				States: xs.States{
					{
						Key:    "active",
						Invoke: []xs.InvokeConfig{{ID: "pong", Logic: pongMachine}},
						// Sends 'PING' event to child machine with ID 'pong'
						Entry: xs.Actions{xs.SendTo("pong", xs.Ev("PING"))},
						On:    map[string]xs.Transitions{"PONG": {{Target: "innerSuccess"}}},
					},
					{Key: "innerSuccess", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Target: "success"}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	sig := newSignal()
	actor := xs.CreateActor(pingMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > with machines > should emit onSnapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2590
func TestInvoke_WithMachines_ShouldEmitOnSnapshot(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", After: map[string]xs.Transitions{"10": {{Target: "b"}}}},
			{Key: "b"},
		},
	})
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Src: "childMachine",
			OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				s, ok := invoke2SnapshotOf(a.Event).(*xs.MachineSnapshot[any])
				if ok && s.Value == "b" {
					sig.Resolve()
				}
			})}}},
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"childMachine": childMachine}})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// invoke3OneTwoCtx mirrors `context: { one?: string; two?: string }` (JS L2633, L2694).
type invoke3OneTwoCtx struct {
	One string
	Two string
}

// invoke3AssignOne mirrors `assign({ one: 'one' })`.
func invoke3AssignOne() xs.Action {
	return xs.Assign(func(a xs.AssignArgs[invoke3OneTwoCtx]) invoke3OneTwoCtx {
		c := a.Context
		c.One = "one"
		return c
	})
}

// invoke3AssignTwo mirrors `assign({ two: 'two' })`.
func invoke3AssignTwo() xs.Action {
	return xs.Assign(func(a xs.AssignArgs[invoke3OneTwoCtx]) invoke3OneTwoCtx {
		c := a.Context
		c.Two = "two"
		return c
	})
}

// invoke3SendBack mirrors `fromCallback(({ sendBack }) => sendBack({ type }))`.
func invoke3SendBack(eventType string) xs.ActorLogic {
	return xs.FromCallback(func(a xs.CallbackArgs) func() {
		a.SendBack(xs.Ev(eventType))
		return nil
	})
}

// JS: invoke > multiple simultaneous services > should start all services at once
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2678
func TestInvoke_MultipleSimultaneousServices_ShouldStartAllServicesAtOnce(t *testing.T) {
	multiple := xs.CreateMachine(xs.MachineConfig[invoke3OneTwoCtx]{
		ID:      "machine",
		Initial: "one",
		Context: invoke3OneTwoCtx{},
		On: map[string]xs.Transitions{
			"ONE": {{Actions: xs.Actions{invoke3AssignOne()}}},
			"TWO": {{Actions: xs.Actions{invoke3AssignTwo()}, Target: ".three"}},
		},
		States: xs.States{
			{
				Key:     "one",
				Initial: "two",
				States: xs.States{
					{
						Key: "two",
						Invoke: []xs.InvokeConfig{
							{ID: "child", Logic: invoke3SendBack("ONE")},
							{ID: "child2", Logic: invoke3SendBack("TWO")},
						},
					},
				},
			},
			{Key: "three", Type: xs.Final},
		},
	})

	sig := newSignal()
	service := xs.CreateActor(multiple)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[invoke3OneTwoCtx]]{
		Complete: func() {
			assert.Equal(t, invoke3OneTwoCtx{One: "one", Two: "two"}, service.GetSnapshot().Context)
			sig.Resolve()
		},
	})

	service.Start()
	sig.Wait(t)
}

// JS: invoke > multiple simultaneous services > should run services in parallel
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2755
func TestInvoke_MultipleSimultaneousServices_ShouldRunServicesInParallel(t *testing.T) {
	parallel := xs.CreateMachine(xs.MachineConfig[invoke3OneTwoCtx]{
		ID:      "machine",
		Initial: "one",
		Context: invoke3OneTwoCtx{},
		On: map[string]xs.Transitions{
			"ONE": {{Actions: xs.Actions{invoke3AssignOne()}}},
			"TWO": {{Actions: xs.Actions{invoke3AssignTwo()}}},
		},
		// allow both invoked services to get a chance to send their events
		// and don't depend on a potential race condition (with an immediate transition)
		After: map[string]xs.Transitions{"10": {{Target: ".three"}}},
		States: xs.States{
			{
				Key:     "one",
				Initial: "two",
				States: xs.States{
					{
						Key:  "two",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "a", Invoke: []xs.InvokeConfig{{ID: "child", Logic: invoke3SendBack("ONE")}}},
							{Key: "b", Invoke: []xs.InvokeConfig{{ID: "child2", Logic: invoke3SendBack("TWO")}}},
						},
					},
				},
			},
			{Key: "three", Type: xs.Final},
		},
	})

	sig := newSignal()
	service := xs.CreateActor(parallel)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[invoke3OneTwoCtx]]{
		Complete: func() {
			assert.Equal(t, invoke3OneTwoCtx{One: "one", Two: "two"}, service.GetSnapshot().Context)
			sig.Resolve()
		},
	})

	service.Start()
	sig.Wait(t)
}

// JS: invoke > multiple simultaneous services > should not invoke an actor if it gets stopped immediately by transitioning away in immediate microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2772
func TestInvoke_MultipleSimultaneousServices_ShouldNotInvokeIfStoppedInImmediateMicrostep(t *testing.T) {
	// Since an actor will be canceled when the state machine leaves the invoking state
	// it does not make sense to start an actor in a state that will be exited immediately
	var actorStarted atomic.Bool

	transientMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "transient",
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID: "doNotInvoke",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						actorStarted.Store(true)
						return nil
					}),
				}},
				Always: xs.Transitions{{Target: "inactive"}},
			},
			{Key: "inactive"},
		},
	})

	service := xs.CreateActor(transientMachine)

	service.Start()

	assert.False(t, actorStarted.Load())
}

// JS: invoke > multiple simultaneous services > should not invoke an actor if it gets stopped immediately by transitioning away in subsequent microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2802
func TestInvoke_MultipleSimultaneousServices_ShouldNotInvokeIfStoppedInSubsequentMicrostep(t *testing.T) {
	// Since an actor will be canceled when the state machine leaves the invoking state
	// it does not make sense to start an actor in a state that will be exited immediately
	var actorStarted atomic.Bool

	transientMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "withNonLeafInvoke",
		States: xs.States{
			{
				Key: "withNonLeafInvoke",
				Invoke: []xs.InvokeConfig{{
					ID: "doNotInvoke",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						actorStarted.Store(true)
						return nil
					}),
				}},
				Initial: "first",
				States: xs.States{
					{Key: "first", Always: xs.Transitions{{Target: "second"}}},
					{Key: "second", Always: xs.Transitions{{Target: "#inactive"}}},
				},
			},
			{Key: "inactive", ID: "inactive"},
		},
	})

	service := xs.CreateActor(transientMachine)

	service.Start()

	assert.False(t, actorStarted.Load())
}

// JS: invoke > multiple simultaneous services > should invoke a service if other service gets stopped in subsequent microstep (#1180)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2840
func TestInvoke_MultipleSimultaneousServices_ShouldInvokeIfOtherServiceStoppedInSubsequentMicrostep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "running",
		States: xs.States{
			{
				Key:  "running",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "one",
						Initial: "active",
						On:      map[string]xs.Transitions{"STOP_ONE": {{Target: ".idle"}}},
						States: xs.States{
							{Key: "idle"},
							{
								Key: "active",
								Invoke: []xs.InvokeConfig{{
									ID: "active",
									Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
										/* ... */
										return nil
									}),
								}},
								On: map[string]xs.Transitions{
									"NEXT": {{Actions: xs.Actions{xs.Raise(xs.Ev("STOP_ONE"))}}},
								},
							},
						},
					},
					{
						Key:     "two",
						Initial: "idle",
						On:      map[string]xs.Transitions{"NEXT": {{Target: ".active"}}},
						States: xs.States{
							{Key: "idle"},
							{
								Key: "active",
								Invoke: []xs.InvokeConfig{{
									ID: "post",
									Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
										return 42, nil
									}),
									OnDone: xs.Transitions{{Target: "#done"}},
								}},
							},
						},
					},
				},
			},
			{Key: "done", ID: "done", Type: xs.Final},
		},
	})

	sig := newSignal()
	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.Ev("NEXT"))
	sig.Wait(t)
}

// JS: invoke > multiple simultaneous services > should invoke an actor when reentering invoking state within a single macrostep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2903
func TestInvoke_MultipleSimultaneousServices_ShouldInvokeWhenReenteringInvokingStateInOneMacrostep(t *testing.T) {
	type ctx struct{ Counter int }
	var actorStartedCount atomic.Int64

	transientMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		Context: ctx{Counter: 0},
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						actorStartedCount.Add(1)
						return nil
					}),
				}},
				Always: xs.Transitions{{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Counter == 0 }),
					Target: "inactive",
				}},
			},
			{
				Key: "inactive",
				// assign({ counter: ({ context }) => ++context.counter })
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Counter++
					return c
				})},
				Always: xs.Transitions{{Target: "active"}},
			},
		},
	})

	service := xs.CreateActor(transientMachine)

	service.Start()

	assert.Equal(t, int64(1), actorStartedCount.Load())
}

// JS: invoke > invoke `src` can be used with invoke `input`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2939
func TestInvoke_SrcCanBeUsedWithInvokeInput(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "searching",
		States: xs.States{
			{
				Key: "searching",
				Invoke: []xs.InvokeConfig{{
					Src:    "search",
					Input:  map[string]any{"endpoint": "example.com"},
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"search": xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (int, error) {
			input, _ := a.Input.(map[string]any)
			assert.Equal(t, "example.com", input["endpoint"])

			return 42, nil
		}),
	}})

	sig := newSignal()
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > invoke `src` can be used with dynamic invoke `input`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L2986
func TestInvoke_SrcCanBeUsedWithDynamicInvokeInput(t *testing.T) {
	type ctx struct{ URL string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "searching",
		Context: ctx{URL: "example.com"},
		States: xs.States{
			{
				Key: "searching",
				Invoke: []xs.InvokeConfig{{
					Src: "search",
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return map[string]any{"endpoint": a.Context.URL}
					}),
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"search": xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (int, error) {
			input, _ := a.Input.(map[string]any)
			assert.Equal(t, "example.com", input["endpoint"])

			return 42, nil
		}),
	}})

	// await new Promise<void>((res) => { ... actor.subscribe({ complete: () => res() }) ... })
	sig := newSignal()
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: invoke > invoke generated ID should be predictable based on the state node where it is defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3036
func TestInvoke_GeneratedIDShouldBePredictableBasedOnDefiningStateNode(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Src: "someSrc",
					OnDone: xs.Transitions{{
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							// invoke ID should not be 'someSrc'
							expectedType := "xstate.done.actor.0.(machine).a"
							assert.Equal(t, expectedType, a.Event.EventType())
							return a.Event.EventType() == expectedType
						}),
						Target: "b",
					}},
				}},
			},
			{Key: "b", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		// fromPromise(() => Promise.resolve())
		"someSrc": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) { return nil, nil }),
	}})

	sig := newSignal()
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// invoke3AssertUniqueChild is the body of the JS it.each at L3078-3119:
// 'invoke config defined as %s should register unique and predictable child in state'.
func invoke3AssertUniqueChild(t *testing.T, invokeConfig xs.InvokeConfig) {
	t.Helper()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "machine",
		Initial: "a",
		States: xs.States{
			{Key: "a", Invoke: []xs.InvokeConfig{invokeConfig}},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"someSrc": xs.FromCallback(func(a xs.CallbackArgs) func() {
			/* ... */
			return nil
		}),
	}})

	assert.NotNil(t, xs.CreateActor(machine).GetSnapshot().Children["0.machine.a"])
}

// JS: invoke > invoke config defined as src with string reference should register unique and predictable child in state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3078
func TestInvoke_InvokeConfigAsSrcWithStringReference_ShouldRegisterUniquePredictableChild(t *testing.T) {
	invoke3AssertUniqueChild(t, xs.InvokeConfig{Src: "someSrc"})
}

// JS: invoke > invoke config defined as src containing a machine directly should register unique and predictable child in state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3078
func TestInvoke_InvokeConfigAsSrcContainingMachine_ShouldRegisterUniquePredictableChild(t *testing.T) {
	invoke3AssertUniqueChild(t, xs.InvokeConfig{Logic: xs.CreateMachine(xs.MachineConfig[any]{ID: "someId"})})
}

// JS: invoke > invoke config defined as src containing a callback actor directly should register unique and predictable child in state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3078
func TestInvoke_InvokeConfigAsSrcContainingCallback_ShouldRegisterUniquePredictableChild(t *testing.T) {
	invoke3AssertUniqueChild(t, xs.InvokeConfig{Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
		/* ... */
		return nil
	})})
}

// https://github.com/statelyai/xstate/issues/464
// JS: invoke > xstate.done.actor events should only select onDone transition on the invoking state when invokee is referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3122
func TestInvoke_DoneActorEventsShouldOnlySelectOnDoneOnInvokingStateWithStringSrc(t *testing.T) {
	var counter atomic.Int64
	var mu sync.Mutex
	invoked := false

	createSingleState := func(key string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "fetch",
			States: xs.States{
				{
					Key: "fetch",
					Invoke: []xs.InvokeConfig{{
						Src:    "fetchSmth",
						OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionRef{Type: "handleSuccess"}}}},
					}},
				},
			},
		}
	}

	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			createSingleState("first"),
			createSingleState("second"),
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"handleSuccess": xs.ActionFunc(func(a xs.ActionArgs[any]) {
				counter.Add(1)
			}),
		},
		Actors: map[string]xs.ActorLogic{
			"fetchSmth": xs.FromPromise(func(c context.Context, _ xs.PromiseArgs) (int, error) {
				mu.Lock()
				if invoked {
					mu.Unlock()
					// create a promise that won't ever resolve for the second invoking state
					<-c.Done()
					return 0, c.Err()
				}
				invoked = true
				mu.Unlock()
				return 42, nil
			}),
		},
	})

	xs.CreateActor(testMachine).Start()

	// JS: `await sleep(0)` so all promise-induced microtasks resolve first. Go
	// promise actors settle on other goroutines, so wait a few ms instead.
	sleep(10)
	assert.Equal(t, int64(1), counter.Load())
}

// JS: invoke > xstate.done.actor events should have unique names when invokee is a machine with an id property
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3176
func TestInvoke_DoneActorEventsShouldHaveUniqueNamesWhenInvokeeIsMachineWithID(t *testing.T) {
	var mu sync.Mutex
	actual := []xs.Event{}

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
						return 42, nil
					}),
					OnDone: xs.Transitions{{Target: "b"}},
				}},
			},
			{Key: "b", Type: xs.Final},
		},
	})

	createSingleState := func(key string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "fetch",
			States: xs.States{
				{Key: "fetch", Invoke: []xs.InvokeConfig{{Logic: childMachine}}},
			},
		}
	}

	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			createSingleState("first"),
			createSingleState("second"),
		},
		On: map[string]xs.Transitions{
			"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				mu.Lock()
				actual = append(actual, a.Event)
				mu.Unlock()
			})}}},
		},
	})

	xs.CreateActor(testMachine).Start()

	// JS: `await sleep(0)` so all promise-induced microtasks resolve first. Go
	// promise actors settle on other goroutines, so wait a few ms instead.
	sleep(10)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []xs.Event{
		xs.DoneActorEvent{ActorID: "0.(machine).first.fetch", Output: nil},
		xs.DoneActorEvent{ActorID: "0.(machine).second.fetch", Output: nil},
	}, actual)
}

// JS: invoke > should get reinstantiated after reentering the invoking state in a microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3242
func TestInvoke_ShouldGetReinstantiatedAfterReenteringInvokingStateInMicrostep(t *testing.T) {
	var invokeCount atomic.Int64

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						invokeCount.Add(1)
						return nil
					}),
				}},
				On: map[string]xs.Transitions{"GO_AWAY_AND_REENTER": {{Target: "b"}}},
			},
			{Key: "b", Always: xs.Transitions{{Target: "a"}}},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("GO_AWAY_AND_REENTER"))

	assert.Equal(t, int64(2), invokeCount.Load())
}

// JS: invoke > invocations should be stopped when the machine reaches done state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3270
func TestInvoke_InvocationsShouldBeStoppedWhenMachineReachesDoneState(t *testing.T) {
	var disposed atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
				return func() {
					disposed.Store(true)
				}
			}),
		}},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FINISH": {{Target: "b"}}}},
			{Key: "b", Type: xs.Final},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("FINISH"))
	assert.True(t, disposed.Load())
}

// JS: invoke > deep invocations should be stopped when the machine reaches done state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3298
func TestInvoke_DeepInvocationsShouldBeStoppedWhenMachineReachesDoneState(t *testing.T) {
	var disposed atomic.Bool
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
				return func() {
					disposed.Store(true)
				}
			}),
		}},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		Invoke:  []xs.InvokeConfig{{Logic: childMachine}},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FINISH": {{Target: "b"}}}},
			{Key: "b", Type: xs.Final},
		},
	})
	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("FINISH"))
	assert.True(t, disposed.Load())
}

// JS: invoke > root invocations should restart on root reentering transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3332
func TestInvoke_RootInvocationsShouldRestartOnRootReenteringTransitions(t *testing.T) {
	var count atomic.Int64

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID: "root",
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				count.Add(1)
				return 42, nil
			}),
		}},
		On: map[string]xs.Transitions{
			"EVENT": {{Target: "#two", Reenter: true}},
		},
		Initial: "one",
		States: xs.States{
			{Key: "one"},
			{Key: "two", ID: "two"},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("EVENT"))

	// JS asserts synchronously: the promise creator runs synchronously in JS,
	// but on its own goroutine in Go (logic.go FromPromise). Wait for the second
	// call, then confirm there is no third one.
	assert.Eventually(t, func() bool { return count.Load() >= 2 }, time.Second, time.Millisecond)
	sleep(10)
	assert.Equal(t, int64(2), count.Load())
}

// JS: invoke > should be able to restart an invoke when reentering the invoking state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3365
func TestInvoke_ShouldRestartInvokeWhenReenteringInvokingState(t *testing.T) {
	var mu sync.Mutex
	actual := []string{}
	invokeCounter := 0

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"ACTIVATE": {{Target: "active"}}}},
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						mu.Lock()
						invokeCounter++
						localID := invokeCounter
						actual = append(actual, "start "+strconv.Itoa(localID))
						mu.Unlock()
						return func() {
							mu.Lock()
							actual = append(actual, "stop "+strconv.Itoa(localID))
							mu.Unlock()
						}
					}),
				}},
				On: map[string]xs.Transitions{
					"REENTER": {{Target: "active", Reenter: true}},
				},
			},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("ACTIVATE"))

	mu.Lock()
	actual = actual[:0]
	mu.Unlock()

	service.Send(xs.Ev("REENTER"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"stop 1", "start 2"}, actual)
}

// JS: invoke > should be able to receive a delayed event sent by the entry action of the invoking state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3410
func TestInvoke_ShouldReceiveDelayedEventSentByEntryActionOfInvokingState(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{
				// sendTo(({ event }) => event.origin, { type: 'PONG' })
				xs.SendTo(xs.NewExpr(func(a xs.ExprArgs[any]) any {
					return a.Event.(xs.E)["origin"]
				}), xs.Ev("PONG")),
			}}},
		},
	})
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key:    "b",
				Invoke: []xs.InvokeConfig{{ID: "foo", Logic: child}},
				// sendTo('foo', ({ self }) => ({ type: 'PING', origin: self }), { delay: 1 })
				Entry: xs.Actions{xs.SendTo("foo", xs.NewExpr(func(a xs.ExprArgs[any]) any {
					return xs.E{"type": "PING", "origin": a.Self}
				}), xs.SendOptions{Delay: ms(1)})},
				On: map[string]xs.Transitions{"PONG": {{Target: "c"}}},
			},
			{Key: "c", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))
	// JS: `await sleep(3)` for a 1ms delay. Go timers fire on other goroutines,
	// so allow a wider margin.
	sleep(10)
	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: invoke input > should provide input to an actor creator
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3458
func TestInvokeInput_ShouldProvideInputToAnActorCreator(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "pending",
		Context: ctx{Count: 42},
		States: xs.States{
			{
				Key: "pending",
				Invoke: []xs.InvokeConfig{{
					Src: "stringService",
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return map[string]any{
							"staticVal": "hello",
							"newCount":  a.Context.Count * 2,
						}
					}),
					OnDone: xs.Transitions{{Target: "success"}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"stringService": xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (bool, error) {
			assert.Equal(t, map[string]any{"newCount": 84, "staticVal": "hello"}, a.Input)

			return true, nil
		}),
	}})

	sig := newSignal()
	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			sig.Resolve()
		},
	})

	service.Start()
	sig.Wait(t)
}

// JS: invoke input > should provide self to input mapper
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invoke.test.ts#L3517
func TestInvokeInput_ShouldProvideSelfToInputMapper(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
				// expect(input.responder.send).toBeDefined(): responder is an ActorRef (has Send)
				input, _ := a.Input.(map[string]any)
				responder, ok := input["responder"].(xs.ActorRef)
				assert.True(t, ok)
				assert.NotNil(t, responder)
				sig.Resolve()
				return nil
			}),
			Input: xs.NewExpr(func(a xs.ExprArgs[any]) any {
				return map[string]any{"responder": a.Self}
			}),
		}},
	})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}
