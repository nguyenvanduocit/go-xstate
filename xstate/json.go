package xstate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

//go:embed machine.schema.json
var machineSchemaJSON []byte

// ValidateMachineSchema mirrors validating a value against XState's JSON
// schema for machine definitions (machine.schema.json) with Ajv: it returns
// the validation errors, or nil when v conforms to the schema.
func ValidateMachineSchema(v any) []string {
	var schema map[string]any
	if err := json.Unmarshal(machineSchemaJSON, &schema); err != nil {
		return []string{err.Error()}
	}
	var doc any
	b, err := json.Marshal(v)
	if err != nil {
		return []string{err.Error()}
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return []string{err.Error()}
	}
	sv := &schemaValidator{root: schema}
	errs := sv.validate(schema, doc, "")
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// schemaValidator implements the JSON Schema (draft-07) keywords used by
// machine.schema.json.
type schemaValidator struct {
	root map[string]any
}

func (sv *schemaValidator) resolveRef(ref string) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var cur any = sv.root
	for _, part := range strings.Split(ref[2:], "/") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	m, _ := cur.(map[string]any)
	return m
}

func jsonType(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		if t == math.Trunc(t) {
			return "integer"
		}
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}

func (sv *schemaValidator) validate(schema map[string]any, v any, path string) []string {
	var errs []string
	fail := func(format string, args ...any) {
		errs = append(errs, path+": "+fmt.Sprintf(format, args...))
	}
	if ref, ok := schema["$ref"].(string); ok {
		target := sv.resolveRef(ref)
		if target == nil {
			fail("unresolved $ref %s", ref)
			return errs
		}
		errs = append(errs, sv.validate(target, v, path)...)
	}
	if t, ok := schema["type"].(string); ok {
		actual := jsonType(v)
		if !(actual == t || (t == "number" && actual == "integer")) {
			fail("must be %s", t)
			return errs
		}
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			if e == v {
				found = true
				break
			}
		}
		if !found {
			fail("must be equal to one of the allowed values")
		}
	}
	if pattern, ok := schema["pattern"].(string); ok {
		if s, isStr := v.(string); isStr {
			if re, err := regexp.Compile(pattern); err == nil && !re.MatchString(s) {
				fail("must match pattern %q", pattern)
			}
		}
	}
	if all, ok := schema["allOf"].([]any); ok {
		for _, sub := range all {
			if m, ok := sub.(map[string]any); ok {
				errs = append(errs, sv.validate(m, v, path)...)
			}
		}
	}
	if one, ok := schema["oneOf"].([]any); ok {
		matches := 0
		for _, sub := range one {
			if m, ok := sub.(map[string]any); ok && len(sv.validate(m, v, path)) == 0 {
				matches++
			}
		}
		if matches != 1 {
			fail("must match exactly one schema in oneOf")
		}
	}
	if obj, ok := v.(map[string]any); ok {
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				if name, _ := r.(string); name != "" {
					if _, has := obj[name]; !has {
						fail("must have required property '%s'", name)
					}
				}
			}
		}
		props, _ := schema["properties"].(map[string]any)
		patternProps, _ := schema["patternProperties"].(map[string]any)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			matched := false
			if ps, ok := props[k].(map[string]any); ok {
				matched = true
				errs = append(errs, sv.validate(ps, obj[k], path+"/"+k)...)
			}
			for pattern, sub := range patternProps {
				if re, err := regexp.Compile(pattern); err == nil && re.MatchString(k) {
					matched = true
					if m, ok := sub.(map[string]any); ok {
						errs = append(errs, sv.validate(m, obj[k], path+"/"+k)...)
					}
				}
			}
			if ap, ok := schema["additionalProperties"].(bool); ok && !ap && !matched {
				fail("must NOT have additional property '%s'", k)
			}
		}
	}
	if arr, ok := v.([]any); ok {
		if minItems, ok := schema["minItems"].(float64); ok && float64(len(arr)) < minItems {
			fail("must NOT have fewer than %v items", minItems)
		}
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range arr {
				errs = append(errs, sv.validate(items, item, fmt.Sprintf("%s/%d", path, i))...)
			}
		}
	}
	return errs
}

// CreateMachineFromJSON mirrors `createMachine(machineObject)` where
// machineObject is a plain JSON object, typically the machine's toJSON()
// output after a JSON round-trip.
func CreateMachineFromJSON(config map[string]any, impl ...Implementations) *StateMachine[any] {
	sc := stateConfigFromJSON("", config)
	mc := MachineConfig[any]{
		ID:          sc.ID,
		Type:        sc.Type,
		Initial:     sc.Initial,
		States:      sc.States,
		Invoke:      sc.Invoke,
		On:          sc.On,
		Entry:       sc.Entry,
		Exit:        sc.Exit,
		Meta:        sc.Meta,
		Output:      sc.Output,
		Tags:        sc.Tags,
		Description: sc.Description,
		initial:     sc.initial,
	}
	if v, ok := config["version"].(string); ok {
		mc.Version = v
	}
	if ctx, ok := config["context"]; ok {
		mc.Context = ctx
	}
	if mc.Type == Compound || mc.Type == Atomic {
		mc.Type = ""
	}
	return CreateMachine(mc, impl...)
}

func stateConfigFromJSON(key string, m map[string]any) StateConfig {
	sc := StateConfig{Key: key}
	sc.ID, _ = m["id"].(string)
	if t, ok := m["type"].(string); ok {
		sc.Type = StateType(t)
	}
	if h, ok := m["history"].(string); ok {
		sc.History = HistoryType(h)
	}
	if d, ok := m["description"].(string); ok {
		sc.Description = d
	}
	sc.Meta = m["meta"]
	sc.Output = m["output"]
	if tags, ok := m["tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				sc.Tags = append(sc.Tags, s)
			}
		}
	}
	switch init := m["initial"].(type) {
	case string:
		sc.Initial = init
	case map[string]any:
		// JS: `stateNode.states[initial.target]` with an array target
		// resolves lazily (and fails) exactly as here.
		var targets []string
		if arr, ok := init["target"].([]any); ok {
			for _, t := range arr {
				if s, ok := t.(string); ok {
					targets = append(targets, s)
				}
			}
		}
		sc.Initial = strings.Join(targets, ",")
		sc.initial = initialTransitionConfig{isObject: true, actions: actionsFromJSON(init["actions"]), meta: init["meta"]}
		if d, ok := init["description"].(string); ok {
			sc.initial.description = d
		}
	}
	sc.Entry = actionsFromJSON(m["entry"])
	sc.Exit = actionsFromJSON(m["exit"])
	if states, ok := m["states"].(map[string]any); ok {
		keys := make([]string, 0, len(states))
		for k := range states {
			keys = append(keys, k)
		}
		order := func(k string) float64 {
			if sm, ok := states[k].(map[string]any); ok {
				if o, ok := sm["order"].(float64); ok {
					return o
				}
			}
			return math.MaxFloat64
		}
		sort.SliceStable(keys, func(i, j int) bool {
			if order(keys[i]) != order(keys[j]) {
				return order(keys[i]) < order(keys[j])
			}
			return keys[i] < keys[j]
		})
		for _, k := range keys {
			if sm, ok := states[k].(map[string]any); ok {
				sc.States = append(sc.States, stateConfigFromJSON(k, sm))
			}
		}
	}
	if on, ok := m["on"].(map[string]any); ok {
		sc.On = map[string]Transitions{}
		for desc, list := range on {
			arr, _ := list.([]any)
			ts := Transitions{}
			for _, item := range arr {
				if tm, ok := item.(map[string]any); ok {
					ts = append(ts, transitionFromJSON(tm))
				}
			}
			sc.On[desc] = ts
		}
	}
	if inv, ok := m["invoke"].([]any); ok {
		for _, item := range inv {
			im, _ := item.(map[string]any)
			if im == nil {
				continue
			}
			ic := InvokeConfig{}
			ic.ID, _ = im["id"].(string)
			ic.Src, _ = im["src"].(string)
			ic.SystemID, _ = im["systemId"].(string)
			ic.Input = im["input"]
			sc.Invoke = append(sc.Invoke, ic)
		}
	}
	return sc
}

func transitionFromJSON(m map[string]any) TransitionConfig {
	tc := TransitionConfig{Actions: actionsFromJSON(m["actions"]), Meta: m["meta"]}
	if arr, ok := m["target"].([]any); ok {
		tc.Targets = []string{}
		for _, t := range arr {
			if s, ok := t.(string); ok {
				tc.Targets = append(tc.Targets, s)
			}
		}
	}
	tc.Reenter, _ = m["reenter"].(bool)
	if d, ok := m["description"].(string); ok {
		tc.Description = d
	}
	if g, ok := m["guard"].(map[string]any); ok {
		typ, _ := g["type"].(string)
		tc.Guard = GuardRef{Type: typ, Params: g["params"]}
	}
	return tc
}

func actionsFromJSON(v any) Actions {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out Actions
	for _, item := range arr {
		switch a := item.(type) {
		case string:
			out = append(out, ActionRef{Type: a})
		case map[string]any:
			typ, _ := a["type"].(string)
			out = append(out, ActionRef{Type: typ, Params: a["params"]})
		}
	}
	return out
}
