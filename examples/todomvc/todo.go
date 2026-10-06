package todomvc

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// TodoContext is todoMachine's extended state (Todo.tsx:10-13).
type TodoContext struct {
	InitialTitle string `json:"initialTitle"`
	Title        string `json:"title"`
}

// TodoInput is todoMachine's input: the todo item the <Todo> component renders.
type TodoInput struct {
	Todo TodoItem `json:"todo"`
}

func inputTodo(input any) TodoItem {
	switch in := input.(type) {
	case TodoInput:
		return in.Todo
	case map[string]any:
		if t, ok := itemOf(in["todo"]); ok {
			return t
		}
	}
	panic("todoMachine: input.todo is required")
}

// TodoMachine mirrors todoMachine (Todo.tsx:8-75). Its `focusInput` and `onCommit`
// actions are the empty functions of the setup() call; Todo.tsx replaces them with
// provide(), see TodoMachineFor.
func TodoMachine() *xs.StateMachine[TodoContext] {
	noop := xs.ActionFunc(func(xs.ActionArgs[TodoContext]) {})
	setup := xs.NewSetup[TodoContext](xs.Implementations{
		Actions: map[string]xs.Action{"focusInput": noop, "onCommit": noop},
	})
	return setup.CreateMachine(xs.MachineConfig[TodoContext]{
		ID:      "todo",
		Initial: "reading",
		ContextFn: func(a xs.ContextArgs) TodoContext {
			title := inputTodo(a.Input).Title
			return TodoContext{InitialTitle: title, Title: title}
		},
		States: xs.States{
			{
				Key: "reading",
				On:  map[string]xs.Transitions{"edit": {{Target: "editing"}}},
			},
			{
				Key: "editing",
				Entry: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[TodoContext]) TodoContext {
						c := a.Context
						c.InitialTitle = c.Title
						return c
					}),
					xs.ActionRef{Type: "focusInput"},
				},
				On: map[string]xs.Transitions{
					"blur": {{Target: "reading", Actions: xs.Actions{xs.ActionRef{Type: "onCommit"}}}},
					"cancel": {{Target: "reading", Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[TodoContext]) TodoContext {
							c := a.Context
							c.Title = c.InitialTitle
							return c
						}),
					}}},
					"change": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[TodoContext]) TodoContext {
							c := a.Context
							c.Title = str(a.Event, "value")
							return c
						}),
					}}},
				},
			},
		},
	})
}

// TodoMachineFor mirrors the useActorRef(todoMachine.provide({...})) call of Todo.tsx:79-100:
// onCommit sends todo.commit to the todos actor with the title currently being edited.
// todo is the item of the first render; like the React closure, it is not refreshed
// when the todos context changes later. focusInput (an input.select() on a timeout)
// is DOM-only and stays the empty function.
func TodoMachineFor(todos xs.ActorRef, todo TodoItem) *xs.StateMachine[TodoContext] {
	return TodoMachine().Provide(xs.Implementations{
		Actions: map[string]xs.Action{
			"onCommit": xs.ActionFunc(func(a xs.ActionArgs[TodoContext]) {
				committed := todo
				committed.Title = a.Context.Title
				todos.Send(xs.E{"type": "todo.commit", "todo": committed})
			}),
		},
	})
}
