package xstate

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---- transitions formatting (stateUtils.ts) ----

func toTransitionConfigArray(ts Transitions) Transitions {
	if ts == nil {
		return Transitions{{}}
	}
	return ts
}

func normalizeTarget(cfg TransitionConfig) []string {
	if cfg.Targets != nil {
		return cfg.Targets
	}
	if cfg.Target == "" {
		return nil
	}
	return []string{cfg.Target}
}

func formatTransition(n *StateNode, descriptor string, cfg TransitionConfig) *TransitionDefinition {
	targets := resolveTarget(n, normalizeTarget(cfg))
	return &TransitionDefinition{
		EventType:   descriptor,
		Source:      n,
		Target:      targets,
		Actions:     append(Actions{}, cfg.Actions...),
		Guard:       cfg.Guard,
		Reenter:     cfg.Reenter,
		Meta:        cfg.Meta,
		Description: cfg.Description,
	}
}

func formatTransitions(n *StateNode) *transitionMap {
	tm := newTransitionMap()
	if n.Config.On != nil {
		for _, descriptor := range sortedKeys(n.Config.On) {
			if descriptor == "" {
				panic(invalidConfig(n.ID, errors.New(`Null events ("") cannot be specified as a transition key. Use `+"`always: { ... }`"+` instead.`)))
			}
			var list []*TransitionDefinition
			for _, t := range toTransitionConfigArray(n.Config.On[descriptor]) {
				list = append(list, formatTransition(n, descriptor, t))
			}
			tm.set(descriptor, orEmpty(list))
		}
	}
	if n.Config.OnDone != nil {
		descriptor := "xstate.done.state." + n.ID
		var list []*TransitionDefinition
		for _, t := range n.Config.OnDone {
			list = append(list, formatTransition(n, descriptor, t))
		}
		tm.set(descriptor, orEmpty(list))
	}
	for _, def := range n.invokeDefs {
		for _, pair := range []struct {
			prefix string
			ts     Transitions
		}{{"xstate.done.actor.", def.OnDone}, {"xstate.error.actor.", def.OnError}, {"xstate.snapshot.", def.OnSnapshot}} {
			if pair.ts == nil {
				continue
			}
			descriptor := pair.prefix + def.ID
			var list []*TransitionDefinition
			for _, t := range pair.ts {
				list = append(list, formatTransition(n, descriptor, t))
			}
			tm.set(descriptor, orEmpty(list))
		}
	}
	for _, d := range n.after {
		existing, _ := tm.get(d.EventType)
		tm.set(d.EventType, append(existing, &d.TransitionDefinition))
	}
	return tm
}

func orEmpty(l []*TransitionDefinition) []*TransitionDefinition {
	if l == nil {
		return []*TransitionDefinition{}
	}
	return l
}

// sortAfterKeys mirrors Object.keys ordering: integer-like keys ascending
// first, then the remaining keys (sorted, since Go maps have no insertion
// order).
func sortAfterKeys(m map[string]Transitions) []string {
	var ints, others []string
	for k := range m {
		if isArrayIndexKey(k) {
			ints = append(ints, k)
		} else {
			others = append(others, k)
		}
	}
	sort.Slice(ints, func(i, j int) bool {
		a, _ := strconv.ParseUint(ints[i], 10, 64)
		b, _ := strconv.ParseUint(ints[j], 10, 64)
		return a < b
	})
	sort.Slice(others, func(i, j int) bool {
		fi, ei := parseJSNumber(others[i])
		fj, ej := parseJSNumber(others[j])
		if ei && ej && fi != fj {
			return fi < fj
		}
		if ei != ej {
			return ei
		}
		return others[i] < others[j]
	})
	return append(ints, others...)
}

func isArrayIndexKey(k string) bool {
	if k == "" || (len(k) > 1 && k[0] == '0') {
		return false
	}
	for _, c := range k {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseJSNumber mirrors `+str` (returns ok=false for NaN).
func parseJSNumber(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return 0, true
	}
	f, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func formatJSNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// getDelayedTransitions mirrors stateUtils.ts getDelayedTransitions; it
// appends the raise/cancel actions of every delay to entry/exit.
func getDelayedTransitions(n *StateNode) []*DelayedTransitionDefinition {
	if n.Config.After == nil {
		return nil
	}
	var out []*DelayedTransitionDefinition
	for _, key := range sortAfterKeys(n.Config.After) {
		var delay any
		var delayRef string
		if f, ok := parseJSNumber(key); ok {
			delay = time.Duration(f * float64(time.Millisecond))
			delayRef = formatJSNumber(f)
		} else {
			delay = key
			delayRef = key
		}
		afterEvent := createAfterEvent(delayRef, n.ID)
		eventType := afterEvent.EventType()
		n.Entry = append(n.Entry, Raise(afterEvent, SendOptions{ID: eventType, Delay: delay}))
		n.Exit = append(n.Exit, Cancel(eventType))
		for _, t := range toTransitionConfigArray(n.Config.After[key]) {
			def := formatTransition(n, eventType, t)
			out = append(out, &DelayedTransitionDefinition{TransitionDefinition: *def, Delay: delay})
		}
	}
	return out
}

func formatInitialTransition(n *StateNode, target string, obj initialTransitionConfig) *TransitionDefinition {
	var resolved *StateNode
	if target != "" {
		resolved = n.States[target]
		if resolved == nil {
			shown := target
			if obj.isObject {
				shown = "[object Object]"
			}
			panic(invalidConfig(n.ID, fmt.Errorf("Initial state node \"%s\" not found on parent state node #%s", shown, n.ID)))
		}
	}
	t := &TransitionDefinition{
		Source:      n,
		Actions:     append(Actions{}, obj.actions...),
		Target:      []*StateNode{},
		Meta:        obj.meta,
		Description: obj.description,
		isInitial:   true,
	}
	if resolved != nil {
		t.Target = []*StateNode{resolved}
	}
	return t
}

func resolveTarget(n *StateNode, targets []string) []*StateNode {
	if targets == nil {
		return nil
	}
	out := make([]*StateNode, 0, len(targets))
	for _, target := range targets {
		out = append(out, resolveOneTarget(n, target))
	}
	return out
}

func resolveOneTarget(n *StateNode, target string) *StateNode {
	if isStateID(target) {
		return n.machine.getStateNodeByID(target)
	}
	isInternal := len(target) > 0 && target[0] == '.'
	if isInternal && n.Parent == nil {
		return getStateNodeByPath(n, target[1:])
	}
	resolved := target
	if isInternal {
		resolved = n.Key + target
	}
	if n.Parent == nil {
		panic(invalidConfig(n.ID, fmt.Errorf("Invalid target: \"%s\" is not a valid target from the root node. Did you mean \".%s\"?", target, target)))
	}
	var result *StateNode
	func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(*ConfigError); ok {
					panic(invalidConfig(n.ID, fmt.Errorf("Invalid transition definition for state node '%s':\n%w", n.ID, err)))
				}
				panic(r)
			}
		}()
		result = getStateNodeByPath(n.Parent, resolved)
	}()
	return result
}

// errorMessage mirrors `err.message`.
func errorMessage(r any) string {
	switch v := r.(type) {
	case error:
		return v.Error()
	case string:
		return v
	}
	return fmt.Sprint(r)
}

// toStatePath mirrors utils.ts toStatePath (handles `\` escapes).
func toStatePath(stateID string) []string {
	var result []string
	var seg strings.Builder
	for i := 0; i < len(stateID); i++ {
		c := stateID[i]
		switch c {
		case '\\':
			if i+1 < len(stateID) {
				seg.WriteByte(stateID[i+1])
			} else {
				seg.WriteString("undefined")
			}
			i++
			continue
		case '.':
			result = append(result, seg.String())
			seg.Reset()
			continue
		}
		seg.WriteByte(c)
	}
	return append(result, seg.String())
}

func getStateNodeChild(n *StateNode, key string) *StateNode {
	if isStateID(key) {
		return n.machine.getStateNodeByID(key)
	}
	result := n.States[key]
	if result == nil {
		panic(invalidConfig(n.ID, fmt.Errorf("Child state '%s' does not exist on '%s'", key, n.ID)))
	}
	return result
}

// getStateNodeByPath mirrors stateUtils.ts getStateNodeByPath for a string path.
func getStateNodeByPath(n *StateNode, statePath string) *StateNode {
	if isStateID(statePath) {
		var found *StateNode
		func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(*ConfigError); !ok {
						panic(r)
					}
				}
			}()
			found = n.machine.getStateNodeByID(statePath)
		}()
		if found != nil {
			return found
		}
	}
	return getStateNodeByPathArray(n, toStatePath(statePath))
}

func getStateNodeByPathArray(n *StateNode, path []string) *StateNode {
	current := n
	for _, key := range path {
		if key == "" {
			break
		}
		current = getStateNodeChild(current, key)
	}
	return current
}
