// Package todomvc ports references/xstate/examples/todomvc-react/src/todosMachine.ts, the
// machine and pure logic of Todo.tsx and the state derivations of Todos.tsx.
package todomvc

import (
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TodoItem mirrors the TodoItem interface of todosMachine.ts.
type TodoItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// Filter values of TodosFilter: "all", "active", "completed". The machine does not
// validate them (filter.change stores whatever string it receives).
const (
	FilterAll       = "all"
	FilterActive    = "active"
	FilterCompleted = "completed"
)

// Context is todosMachine's extended state.
type Context struct {
	Todo   string     `json:"todo"`
	Todos  []TodoItem `json:"todos"`
	Filter string     `json:"filter"`
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

// itemOf reads the `todo` payload of todo.commit: a TodoItem, or the decoded-JSON
// object a trace sends.
func itemOf(v any) (TodoItem, bool) {
	switch t := v.(type) {
	case TodoItem:
		return t, true
	case *TodoItem:
		return *t, t != nil
	case map[string]any:
		id, _ := t["id"].(string)
		title, _ := t["title"].(string)
		completed, _ := t["completed"].(bool)
		return TodoItem{ID: id, Title: title, Completed: completed}, true
	}
	return TodoItem{}, false
}

func str(e xs.Event, key string) string {
	s, _ := e.(xs.E)[key].(string)
	return s
}

// mapTodos returns a copy of todos with fn applied to each item.
func mapTodos(todos []TodoItem, fn func(TodoItem) TodoItem) []TodoItem {
	out := make([]TodoItem, 0, len(todos))
	for _, t := range todos {
		out = append(out, fn(t))
	}
	return out
}

// filterBy returns the items for which keep is true (never nil, so it marshals as []).
func filterBy(todos []TodoItem, keep func(TodoItem) bool) []TodoItem {
	out := make([]TodoItem, 0, len(todos))
	for _, t := range todos {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

func assignTodos(fn func(c Context, e xs.Event) []TodoItem) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		c := a.Context
		c.Todos = fn(a.Context, a.Event)
		return c
	})
}

// Machine mirrors todosMachine. newID replaces the Math.random().toString(36).substring(7)
// expression of the newTodo.commit action (todosMachine.ts:46).
func Machine(newID func() string) *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID: "todos",
		Context: Context{
			Todo:   "",
			Todos:  []TodoItem{{ID: "1", Title: "Learn state machines", Completed: false}},
			Filter: FilterAll,
		},
		On: map[string]xs.Transitions{
			"newTodo.change": {{Actions: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[Context]) Context {
					c := a.Context
					c.Todo = str(a.Event, "value")
					return c
				}),
			}}},
			"newTodo.commit": {{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
					return len(jsTrim(str(a.Event, "value"))) > 0
				}),
				Actions: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[Context]) Context {
						c := a.Context
						c.Todo = ""
						c.Todos = append(append([]TodoItem{}, a.Context.Todos...), TodoItem{
							ID:        newID(),
							Title:     str(a.Event, "value"),
							Completed: false,
						})
						return c
					}),
				},
			}},
			"todo.commit": {{Actions: xs.Actions{
				assignTodos(func(c Context, e xs.Event) []TodoItem {
					update, _ := itemOf(e.(xs.E)["todo"])
					if len(jsTrim(update.Title)) == 0 {
						return filterBy(c.Todos, func(t TodoItem) bool { return t.ID != update.ID })
					}
					return mapTodos(c.Todos, func(t TodoItem) TodoItem {
						if t.ID == update.ID {
							return update
						}
						return t
					})
				}),
			}}},
			"todo.delete": {{Actions: xs.Actions{
				assignTodos(func(c Context, e xs.Event) []TodoItem {
					id := str(e, "id")
					return filterBy(c.Todos, func(t TodoItem) bool { return t.ID != id })
				}),
			}}},
			"filter.change": {{Actions: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[Context]) Context {
					c := a.Context
					c.Filter = str(a.Event, "filter")
					return c
				}),
			}}},
			"todo.mark": {{Actions: xs.Actions{
				assignTodos(func(c Context, e xs.Event) []TodoItem {
					id, mark := str(e, "id"), str(e, "mark")
					return mapTodos(c.Todos, func(t TodoItem) TodoItem {
						if t.ID == id {
							t.Completed = mark == "completed"
						}
						return t
					})
				}),
			}}},
			"todo.markAll": {{Actions: xs.Actions{
				assignTodos(func(c Context, e xs.Event) []TodoItem {
					mark := str(e, "mark")
					return mapTodos(c.Todos, func(t TodoItem) TodoItem {
						t.Completed = mark == "completed"
						return t
					})
				}),
			}}},
			"todos.clearCompleted": {{Actions: xs.Actions{
				assignTodos(func(c Context, _ xs.Event) []TodoItem {
					return filterBy(c.Todos, func(t TodoItem) bool { return !t.Completed })
				}),
			}}},
		},
	})
}
