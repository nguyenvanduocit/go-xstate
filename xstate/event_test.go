package xstate_test

import (
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: events > should be able to respond to sender by sending self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/event.test.ts#L10
func TestEvent_ShouldBeAbleToRespondToSenderBySendingSelf(t *testing.T) {
	sig := newSignal()

	authServerMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "authServer",
		Initial: "waitingForCode",
		States: xs.States{
			{Key: "waitingForCode", On: map[string]xs.Transitions{
				"CODE": {{Actions: xs.Actions{
					xs.SendTo(
						xs.NewExpr(func(a xs.ExprArgs[any]) any {
							sender := a.Event.(xs.E)["sender"]
							assert.NotNil(t, sender)
							return sender
						}),
						xs.Ev("TOKEN"),
						xs.SendOptions{Delay: 10 * time.Millisecond},
					),
				}}},
			}},
		},
	})

	authClientMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "authClient",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{
				"AUTH": {{Target: "authorizing"}},
			}},
			{
				Key:    "authorizing",
				Invoke: []xs.InvokeConfig{{ID: "auth-server", Logic: authServerMachine}},
				Entry: xs.Actions{
					xs.SendTo("auth-server", xs.NewExpr(func(a xs.ExprArgs[any]) any {
						return xs.E{"type": "CODE", "sender": a.Self}
					})),
				},
				On: map[string]xs.Transitions{
					"TOKEN": {{Target: "authorized"}},
				},
			},
			{Key: "authorized", Type: xs.Final},
		},
	})

	service := xs.CreateActor(authClientMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.Ev("AUTH"))

	sig.Wait(t)
}

// JS: nested transitions > only take the transition of the most inner matching event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/event.test.ts#L73
func TestEvent_NestedTransitions_OnlyTakeTheTransitionOfTheMostInnerMatchingEvent(t *testing.T) {
	type signInContext struct {
		Email    string
		Password string
	}

	authMachine := xs.CreateMachine(xs.MachineConfig[signInContext]{
		Context: signInContext{Email: "", Password: ""},
		Initial: "passwordField",
		States: xs.States{
			{
				Key:     "passwordField",
				Initial: "hidden",
				States: xs.States{
					{Key: "hidden", On: map[string]xs.Transitions{
						// We want to assign the new password but remain in the hidden
						// state
						"changePassword": {{Actions: xs.Actions{xs.ActionRef{Type: "assignPassword"}}}},
					}},
					{Key: "valid"},
					{Key: "invalid"},
				},
				On: map[string]xs.Transitions{
					"changePassword": {
						{
							Guard: xs.GuardFunc(func(a xs.GuardArgs[signInContext]) bool {
								return len(a.Event.(xs.E)["password"].(string)) >= 10
							}),
							Target:  ".invalid",
							Actions: xs.Actions{xs.ActionRef{Type: "assignPassword"}},
						},
						{
							Target:  ".valid",
							Actions: xs.Actions{xs.ActionRef{Type: "assignPassword"}},
						},
					},
				},
			},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"assignPassword": xs.Assign(func(a xs.AssignArgs[signInContext]) signInContext {
				c := a.Context
				c.Password = a.Event.(xs.E)["password"].(string)
				return c
			}),
		},
	})
	password := "xstate123"
	actorRef := xs.CreateActor(authMachine).Start()
	actorRef.Send(xs.E{"type": "changePassword", "password": password})

	snapshot := actorRef.GetSnapshot()
	assert.Equal(t, map[string]any{"passwordField": "hidden"}, snapshot.Value)
	assert.Equal(t, signInContext{Password: password, Email: ""}, snapshot.Context)
}
