package friends

import (
	"math"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// FriendsContext is friendsMachine's extended state. Friends holds the spawned
// friendMachine actors; they marshal as {"xstate$$type":1,"id":...} like JS.
type FriendsContext struct {
	NewFriendName string        `json:"newFriendName"`
	Friends       []xs.ActorRef `json:"friends"`
}

// jsTrim mirrors String.prototype.trim: ECMAScript WhiteSpace and LineTerminator.
// It differs from strings.TrimSpace on U+0085 (trimmed by Go, not by JS) and
// U+FEFF (trimmed by JS, not by Go).
func jsTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
			return true
		}
		return r >= 0x2000 && r <= 0x200A
	})
}

// friendAt mirrors context.friends[event.index]: nil when index is not an
// integer inside the list (JS yields undefined).
func friendAt(friends []xs.ActorRef, index any) xs.ActorRef {
	i, ok := indexOf(index)
	if !ok || i < 0 || i >= len(friends) {
		return nil
	}
	return friends[i]
}

func indexOf(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		if n != math.Trunc(n) || math.IsInf(n, 0) {
			return 0, false
		}
		return int(n), true
	}
	return 0, false
}

// FriendsMachine mirrors friendsMachine. makeID replaces the module-level
// makeId() (Math.random().toString(36).substring(7)); the spawned actor id is
// "friend-" + makeID().
func FriendsMachine(makeID func() string) *xs.StateMachine[FriendsContext] {
	friend := FriendMachine()
	return xs.CreateMachine(xs.MachineConfig[FriendsContext]{
		ID:      "friends",
		Context: FriendsContext{NewFriendName: "", Friends: []xs.ActorRef{}},
		On: map[string]xs.Transitions{
			"NEW_FRIEND.CHANGE": {{Actions: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[FriendsContext]) FriendsContext {
					c := a.Context
					c.NewFriendName, _ = a.Event.(xs.E)["name"].(string)
					return c
				}),
			}}},
			"FRIENDS.ADD": {{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[FriendsContext]) bool {
					name, _ := a.Event.(xs.E)["name"].(string)
					return len(jsTrim(name)) > 0
				}),
				Actions: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[FriendsContext]) FriendsContext {
						c := a.Context
						child := a.Spawn(friend, xs.SpawnOptions{
							ID:    "friend-" + makeID(),
							Input: FriendInput{Name: c.NewFriendName},
						})
						c.Friends = append(append([]xs.ActorRef{}, c.Friends...), child)
						c.NewFriendName = ""
						return c
					}),
				},
			}},
			"FRIEND.REMOVE": {{Actions: xs.Actions{
				// Stop the friend actor to unsubscribe
				xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[FriendsContext]) any {
					if ref := friendAt(a.Context.Friends, a.Event.(xs.E)["index"]); ref != nil {
						return ref
					}
					return nil
				})),
				// Remove the friend from the list by index
				xs.Assign(func(a xs.AssignArgs[FriendsContext]) FriendsContext {
					c := a.Context
					remove, hasRemove := indexOf(a.Event.(xs.E)["index"])
					kept := []xs.ActorRef{}
					for i, f := range c.Friends {
						if !hasRemove || i != remove {
							kept = append(kept, f)
						}
					}
					c.Friends = kept
					return c
				}),
			}}},
		},
	})
}
