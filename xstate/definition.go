package xstate

import (
	"strconv"
)

// TransitionDefinition mirrors a normalized transition.
type TransitionDefinition struct {
	EventType   string
	Source      *StateNode
	Target      []*StateNode // nil for targetless transitions
	Actions     Actions
	Guard       Guard
	Reenter     bool
	Meta        any
	Description string

	isInitial bool
}

// ToJSON mirrors transitionDefinition.toJSON().
func (t *TransitionDefinition) ToJSON() map[string]any {
	out := map[string]any{
		"eventType": t.EventType,
		"actions":   serializeActions(t.Actions),
		"reenter":   t.Reenter,
		"source":    "#" + t.Source.ID,
	}
	if t.isInitial {
		out["eventType"] = nil
	}
	if t.Guard != nil {
		out["guard"] = serializeGuard(t.Guard)
	}
	if t.Target != nil {
		targets := make([]string, len(t.Target))
		for i, tn := range t.Target {
			targets[i] = "#" + tn.ID
		}
		out["target"] = targets
	}
	if t.Meta != nil {
		out["meta"] = t.Meta
	}
	if t.Description != "" {
		out["description"] = t.Description
	}
	return out
}

// MarshalJSON serializes the transition with ToJSON.
func (t *TransitionDefinition) MarshalJSON() ([]byte, error) { return jsonMarshal(t.ToJSON()) }

// DelayedTransitionDefinition mirrors an entry of stateNode.after.
type DelayedTransitionDefinition struct {
	TransitionDefinition
	Delay any // time.Duration or delay name
}

// InvokeDefinition mirrors a normalized invoke.
type InvokeDefinition struct {
	ID         string
	SystemID   string
	Src        any // string name or ActorLogic
	Input      any
	OnDone     Transitions
	OnError    Transitions
	OnSnapshot Transitions
}

// ToJSON mirrors invokeDefinition.toJSON().
func (d *InvokeDefinition) ToJSON() map[string]any {
	out := map[string]any{"type": "xstate.invoke", "src": d.Src, "id": d.ID}
	if d.SystemID != "" {
		out["systemId"] = d.SystemID
	}
	if d.Input != nil {
		if _, isExpr := d.Input.(Expr); !isExpr {
			out["input"] = d.Input
		}
	}
	return out
}

// MarshalJSON serializes the invoke definition with ToJSON.
func (d *InvokeDefinition) MarshalJSON() ([]byte, error) { return jsonMarshal(d.ToJSON()) }

func createInvokeID(stateNodeID string, index int) string {
	return strconv.Itoa(index) + "." + stateNodeID
}

func (n *StateNode) computeInvoke() []*InvokeDefinition {
	defs := make([]*InvokeDefinition, 0, len(n.Config.Invoke))
	for i, cfg := range n.Config.Invoke {
		id := cfg.ID
		if id == "" {
			id = createInvokeID(n.ID, i)
		}
		src := cfg.Src
		if src == "" {
			src = "xstate.invoke." + createInvokeID(n.ID, i)
		}
		defs = append(defs, &InvokeDefinition{
			ID:         id,
			SystemID:   cfg.SystemID,
			Src:        src,
			Input:      cfg.Input,
			OnDone:     cfg.OnDone,
			OnError:    cfg.OnError,
			OnSnapshot: cfg.OnSnapshot,
		})
	}
	return defs
}

// Definition mirrors stateNode.definition.
func (n *StateNode) Definition() StateNodeDefinition {
	var initial any
	if n.initialErr == nil && n.initial != nil && (len(n.initial.Target) > 0 || n.Type == Compound) {
		targets := make([]string, len(n.initial.Target))
		for i, t := range n.initial.Target {
			targets[i] = "#" + t.ID
		}
		init := map[string]any{
			"target":    targets,
			"source":    "#" + n.ID,
			"actions":   serializeActions(n.initial.Actions),
			"eventType": nil,
			"reenter":   false,
		}
		if n.initial.Meta != nil {
			init["meta"] = n.initial.Meta
		}
		if n.initial.Description != "" {
			init["description"] = n.initial.Description
		}
		initial = init
	}
	states := map[string]any{}
	for _, k := range n.childOrder {
		states[k] = n.States[k].Definition()
	}
	on := map[string]any{}
	var transitions []any
	for _, k := range n.transitions.keys {
		ts := n.transitions.m[k]
		if len(ts) > 0 {
			list := make([]any, len(ts))
			for i, t := range ts {
				list[i] = t.ToJSON()
			}
			on[k] = list
		}
		for _, t := range ts {
			transitions = append(transitions, t.ToJSON())
		}
	}
	if transitions == nil {
		transitions = []any{}
	}
	invoke := make([]any, len(n.invokeDefs))
	for i, d := range n.invokeDefs {
		invoke[i] = d.ToJSON()
	}
	var history any = false
	if n.Type == History {
		history = string(n.History)
	}
	order := n.Order
	if order == 0 {
		order = -1
	}
	def := map[string]any{
		"id":          n.ID,
		"key":         n.Key,
		"version":     n.machineVersion(),
		"type":        string(n.Type),
		"initial":     initial,
		"history":     history,
		"states":      states,
		"on":          on,
		"transitions": transitions,
		"entry":       serializeActions(n.Entry),
		"exit":        serializeActions(n.Exit),
		"meta":        n.Meta,
		"order":       order,
		"output":      serializeValue(n.Output),
		"invoke":      invoke,
		"description": n.Description,
		"tags":        append([]string{}, n.Tags...),
	}
	if n.Description == "" {
		def["description"] = nil
	}
	// JSON.stringify drops undefined properties.
	for k, v := range def {
		if v == nil {
			delete(def, k)
		}
	}
	return def
}

func (n *StateNode) machineVersion() any {
	if v := n.machine.machineVersion(); v != "" {
		return v
	}
	return nil
}

// MarshalJSON serializes the state node as its definition (JS toJSON).
func (n *StateNode) MarshalJSON() ([]byte, error) { return jsonMarshal(n.Definition()) }

func serializeValue(v any) any {
	if _, ok := v.(Expr); ok {
		return nil
	}
	return v
}

func serializeActions(actions Actions) []any {
	out := make([]any, 0, len(actions))
	for _, a := range actions {
		out = append(out, serializeAction(a))
	}
	return out
}

// serializeAction mirrors toSerializableAction.
func serializeAction(a Action) any {
	switch act := a.(type) {
	case ActionRef:
		m := map[string]any{"type": act.Type}
		if act.Params != nil {
			m["params"] = serializeValue(act.Params)
		}
		return m
	case *ActionRef:
		return serializeAction(*act)
	case *builtinAction:
		return map[string]any{"type": act.typ}
	case *inlineAction:
		return map[string]any{"type": ""}
	}
	return map[string]any{"type": a.ActionType()}
}

func serializeGuard(g Guard) any {
	switch gd := g.(type) {
	case GuardRef:
		m := map[string]any{"type": gd.Type}
		if gd.Params != nil {
			m["params"] = serializeValue(gd.Params)
		}
		return m
	}
	return map[string]any{"type": g.GuardType()}
}
