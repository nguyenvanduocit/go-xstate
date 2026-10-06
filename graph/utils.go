package graph

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// jsonStringify mirrors JSON.stringify(value) (simpleStringify): no HTML
// escaping, and function-valued map entries are dropped like JS drops
// function-valued properties. It panics when the value cannot be encoded
// (JS: JSON.stringify throws).
func jsonStringify(value any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(jsonValue(reflect.ValueOf(value))); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// jsonStringifyEvent mirrors JSON.stringify(event). An xs.E is written with
// `type` first, as in the `{ type, ...payload }` literal it stands for; the
// other keys follow in JS object key order (see jsObjectKeyOrder).
func jsonStringifyEvent(event xs.Event) string {
	e, ok := event.(xs.E)
	if !ok {
		return jsonStringify(event)
	}
	var rest []string
	for k := range e {
		if k != "type" {
			rest = append(rest, k)
		}
	}
	slices.Sort(rest)
	keys := rest
	if _, hasType := e["type"]; hasType {
		keys = append([]string{"type"}, rest...)
	}
	var b strings.Builder
	b.WriteString("{")
	for _, k := range jsObjectKeyOrder(keys) {
		if unencodable(reflect.ValueOf(e[k])) {
			continue
		}
		if b.Len() > 1 {
			b.WriteString(",")
		}
		b.WriteString(jsonStringify(k) + ":" + jsonStringify(e[k]))
	}
	b.WriteString("}")
	return b.String()
}

// stateValueJSON mirrors JSON.stringify(snapshot.value): JS builds the value
// object in state node document order, then enumerates it in JS object key
// order. node is the state node the value belongs to (the root for a
// snapshot value); unknown keys follow sorted.
func stateValueJSON(value xs.StateValue, node *xs.StateNode) string {
	m, ok := value.(map[string]any)
	if !ok || node == nil {
		return jsonStringify(value)
	}
	keys := make([]string, 0, len(m))
	known := map[string]bool{}
	for _, child := range node.ChildStates() {
		if _, ok := m[child.Key]; ok {
			keys = append(keys, child.Key)
			known[child.Key] = true
		}
	}
	var rest []string
	for k := range m {
		if !known[k] {
			rest = append(rest, k)
		}
	}
	slices.Sort(rest)
	keys = jsObjectKeyOrder(append(keys, rest...))
	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(jsonStringify(k) + ":" + stateValueJSON(m[k], node.States[k]))
	}
	b.WriteString("}")
	return b.String()
}

// rootNodeOf returns the root state node among a snapshot's active nodes.
func rootNodeOf(nodes []*xs.StateNode) *xs.StateNode {
	for _, n := range nodes {
		if n.Parent == nil {
			return n
		}
	}
	return nil
}

// appendJSKey mirrors adding a new key to a JS object: array-index keys
// enumerate first in ascending numeric order, other keys in insertion order.
func appendJSKey(keys []string, key string) []string {
	n, ok := arrayIndex(key)
	if !ok {
		return append(keys, key)
	}
	i := 0
	for i < len(keys) {
		m, isIndex := arrayIndex(keys[i])
		if !isIndex || m > n {
			break
		}
		i++
	}
	return slices.Insert(keys, i, key)
}

// jsObjectKeyOrder returns keys (given in insertion order) in JS object
// enumeration order.
func jsObjectKeyOrder(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = appendJSKey(out, k)
	}
	return out
}

// arrayIndex reports whether key is a canonical JS array index ("0", "42";
// not "01" or "-1") and returns its value.
func arrayIndex(key string) (uint64, bool) {
	if key == "" || (len(key) > 1 && key[0] == '0') {
		return 0, false
	}
	n, err := strconv.ParseUint(key, 10, 32)
	if err != nil || n == math.MaxUint32 {
		return 0, false
	}
	return n, true
}

// jsonValue rewrites string-keyed maps and slices of v so that entries JSON
// cannot encode (functions, channels) are dropped, as JSON.stringify does.
func jsonValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Interface {
			return jsonValue(v.Elem())
		}
	case reflect.Map:
		if v.IsNil() || v.Type().Key().Kind() != reflect.String {
			break
		}
		if _, custom := v.Interface().(json.Marshaler); custom {
			break
		}
		out := make(map[string]any, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if unencodable(iter.Value()) {
				continue
			}
			out[iter.Key().String()] = jsonValue(iter.Value())
		}
		return out
	case reflect.Slice:
		if v.IsNil() || v.Type().Elem().Kind() == reflect.Uint8 {
			break
		}
		if _, custom := v.Interface().(json.Marshaler); custom {
			break
		}
		out := make([]any, v.Len())
		for i := range out {
			if unencodable(v.Index(i)) {
				continue // JS: functions in arrays become null
			}
			out[i] = jsonValue(v.Index(i))
		}
		return out
	}
	return v.Interface()
}

func unencodable(v reflect.Value) bool {
	for v.IsValid() && v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return true
	}
	return false
}

// jsonHasKeys mirrors `Object.keys(value ?? {}).length > 0` for a value as
// JSON sees it.
func jsonHasKeys(value any) bool {
	s := jsonStringify(value)
	switch {
	case strings.HasPrefix(s, "{"):
		var m map[string]json.RawMessage
		return json.Unmarshal([]byte(s), &m) == nil && len(m) > 0
	case strings.HasPrefix(s, "["):
		var a []json.RawMessage
		return json.Unmarshal([]byte(s), &a) == nil && len(a) > 0
	case strings.HasPrefix(s, `"`):
		return s != `""`
	}
	return false
}

// isZero reports whether s is the zero value of S (JS `undefined` for an
// absent snapshot).
func isZero[S any](s S) bool {
	return reflect.ValueOf(&s).Elem().IsZero()
}

// sameState mirrors `a === b` for snapshots (reference identity for the
// pointer snapshots logic produces).
func sameState[S any](a, b S) bool {
	va, vb := reflect.ValueOf(any(a)), reflect.ValueOf(any(b))
	if !va.IsValid() || !vb.IsValid() {
		return va.IsValid() == vb.IsValid()
	}
	if va.Type() != vb.Type() || !va.Comparable() || !vb.Comparable() {
		return false
	}
	return va.Equal(vb)
}

// formatPathTestResult mirrors formatPathTestResult(path, testPathResult,
// options) with the default (identity) formatColor, so the pass/fail state of
// each step does not change the text.
func formatPathTestResult[S xs.Snapshot](path StatePath[S], testPathResult TestPathResult[S], serializeState func(S, xs.Event, S) string, serializeEvent func(xs.Event) string) string {
	if serializeState == nil {
		serializeState = func(state S, _ xs.Event, _ S) string { return jsonStringify(state) }
	}
	if serializeEvent == nil {
		serializeEvent = jsonStringifyEvent
	}
	var none S

	var lastEvent xs.Event
	if len(path.Steps) > 0 {
		lastEvent = path.Steps[len(path.Steps)-1].Event
	}
	targetStateString := serializeState(path.State, lastEvent, none)

	parts := make([]string, 0, len(testPathResult.Steps)+1)
	for i, s := range testPathResult.Steps {
		var prevEvent xs.Event
		if i > 0 {
			prevEvent = testPathResult.Steps[i-1].Step.Event
		}
		stateString := serializeState(s.Step.State, prevEvent, none)
		eventString := serializeEvent(s.Step.Event)

		parts = append(parts, "\tState: "+stateString+"\n\tEvent: "+eventString)
	}
	parts = append(parts, "\tState: "+targetStateString)

	return "\nPath:\n" + strings.Join(parts, "\n\n")
}

// metaDescriber is implemented by TestMeta: it returns the description text
// and whether it is a plain description (quoted) rather than the result of a
// description function (used verbatim).
type metaDescriber interface {
	describe(snapshot any) (text string, quoted bool)
}

// getDescription mirrors getDescription(snapshot).
func getDescription(snapshot machineSnapshotView) string {
	j := snapshot.ToJSON()
	contextString := ""
	if jsonHasKeys(j["context"]) {
		contextString = "(" + jsonStringify(j["context"]) + ")"
	}

	meta := snapshot.GetMeta()
	var stateStrings []string
	for _, sn := range snapshot.StateNodes() {
		if sn.Type != xs.Atomic && sn.Type != xs.Final {
			continue
		}
		m := meta[sn.ID]
		if m == nil {
			stateStrings = append(stateStrings, `"`+strings.Join(sn.Path, ".")+`"`)
			continue
		}

		text, quoted := "", true
		switch d := m.(type) {
		case metaDescriber:
			text, quoted = d.describe(snapshot)
		case map[string]any:
			text, _ = d["description"].(string)
		}
		switch {
		case !quoted:
			stateStrings = append(stateStrings, text)
		case text != "":
			stateStrings = append(stateStrings, `"`+text+`"`)
		default:
			stateStrings = append(stateStrings, stateValueJSON(j["value"], rootNodeOf(snapshot.StateNodes())))
		}
	}

	plural := "s"
	if len(stateStrings) == 1 {
		plural = ""
	}
	return "state" + plural + " " + strings.Join(stateStrings, ", ") + strings.TrimSpace(" "+contextString)
}

// formatEvent mirrors TestModel's formatEvent(event): the type, followed by
// the JSON of the remaining properties when there are any.
func formatEvent(event xs.Event) string {
	other := eventProperties(event)

	propertyString := ""
	if len(other) > 0 {
		propertyString = " (" + jsonStringify(other) + ")"
	}

	return event.EventType() + propertyString
}

// eventProperties mirrors `const { type, ...other } = event`.
func eventProperties(event xs.Event) map[string]any {
	if e, ok := event.(xs.E); ok {
		other := make(map[string]any, len(e))
		for k, v := range e {
			if k != "type" {
				other[k] = v
			}
		}
		return other
	}
	var other map[string]any
	if json.Unmarshal([]byte(jsonStringify(event)), &other) != nil {
		return nil
	}
	delete(other, "type")
	return other
}
