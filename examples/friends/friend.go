// Package friends ports references/xstate/examples/friends-list-react/src/friendMachine.ts
// and friendsMachine.ts.
package friends

import (
	"context"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// FriendContext is friendMachine's extended state.
type FriendContext struct {
	PrevName string `json:"prevName"`
	Name     string `json:"name"`
}

// FriendInput is friendMachine's input.
type FriendInput struct {
	Name string `json:"name"`
}

// SaveUser mirrors the saveUser actor: a simulated network request that
// resolves with true after one second.
func SaveUser() xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (bool, error) {
		select {
		case <-time.After(time.Second):
			return true, nil
		case <-ctx.Done():
			return false, context.Cause(ctx)
		}
	})
}

// inputName reads input.name from a FriendInput or from the decoded-JSON form
// of it (the golden file records input as a JSON object).
func inputName(input any) string {
	switch in := input.(type) {
	case FriendInput:
		return in.Name
	case map[string]any:
		name, _ := in["name"].(string)
		return name
	}
	panic("friendMachine: input.name is required")
}

// FriendMachine mirrors friendMachine.
func FriendMachine() *xs.StateMachine[FriendContext] {
	setup := xs.NewSetup[FriendContext](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"saveUser": SaveUser()},
	})
	return setup.CreateMachine(xs.MachineConfig[FriendContext]{
		ID:      "friend",
		Initial: "reading",
		ContextFn: func(a xs.ContextArgs) FriendContext {
			name := inputName(a.Input)
			return FriendContext{PrevName: name, Name: name}
		},
		States: xs.States{
			{
				Key:  "reading",
				Tags: xs.Tags{"read"},
				On: map[string]xs.Transitions{
					"EDIT": {{Target: "editing"}},
				},
			},
			{
				Key:  "editing",
				Tags: xs.Tags{"form"},
				On: map[string]xs.Transitions{
					"SET_NAME": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[FriendContext]) FriendContext {
							c := a.Context
							c.Name, _ = a.Event.(xs.E)["value"].(string)
							return c
						}),
					}}},
					"SAVE": {{Target: "saving"}},
				},
			},
			{
				Key:  "saving",
				Tags: xs.Tags{"form", "saving"},
				Invoke: []xs.InvokeConfig{{
					Src: "saveUser",
					OnDone: xs.Transitions{{
						Target: "reading",
						Actions: xs.Actions{
							xs.Assign(func(a xs.AssignArgs[FriendContext]) FriendContext {
								c := a.Context
								c.PrevName = c.Name
								return c
							}),
						},
					}},
				}},
			},
		},
		On: map[string]xs.Transitions{
			"CANCEL": {{
				Actions: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[FriendContext]) FriendContext {
						c := a.Context
						c.Name = c.PrevName
						return c
					}),
				},
				Target: ".reading",
			}},
		},
	})
}
