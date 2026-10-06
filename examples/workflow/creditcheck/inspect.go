package creditcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// The JS entry prints objects with console.log, which the Bun runtime renders
// as: unquoted identifier keys, double-quoted strings, one property per line,
// two-space indent, trailing commas, `{}` for an empty object. logInspect
// renders the JSON form of a value (struct field order is preserved) the same
// way, for the shapes this example prints: nested objects of strings, numbers
// and nulls.

// Event views keep the property order of the JS event objects.
type (
	initView struct {
		Type  string `json:"type"`
		Input any    `json:"input"`
	}
	resolveView struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}
	doneActorView struct {
		Type    string `json:"type"`
		Output  any    `json:"output"`
		ActorID string `json:"actorId"`
	}
)

// eventView converts an engine event to the shape of its JS object.
func eventView(e xs.Event) any {
	switch ev := e.(type) {
	case xs.InitEvent:
		return initView{ev.EventType(), ev.Input}
	case xs.PromiseResolveEvent:
		return resolveView{ev.EventType(), ev.Data}
	case xs.DoneActorEvent:
		return doneActorView{ev.EventType(), ev.Output, ev.ActorID}
	case xs.E:
		// Dynamic events have no fixed order: type first, then sorted keys.
		keys := make([]string, 0, len(ev))
		for k := range ev {
			if k != "type" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		return orderedMap{keys: append([]string{"type"}, keys...), m: ev}
	}
	return map[string]any{"type": e.EventType()}
}

// orderedMap marshals a map with an explicit key order.
type orderedMap struct {
	keys []string
	m    map[string]any
}

func (o orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, err := json.Marshal(o.m[k])
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// logInspect mirrors console.log(label, value).
func logInspect(w io.Writer, label string, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		fmt.Fprintln(w, label, fmt.Sprint(value))
		return
	}
	var sb strings.Builder
	dec := json.NewDecoder(bytes.NewReader(raw))
	writeValue(&sb, dec, 0)
	fmt.Fprintln(w, label, sb.String())
}

// writeValue renders the next JSON value of dec at the given nesting depth.
func writeValue(sb *strings.Builder, dec *json.Decoder, depth int) {
	tok, _ := dec.Token()
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			writeObject(sb, dec, depth)
		} else {
			writeArray(sb, dec, depth)
		}
	case string:
		b, _ := json.Marshal(t)
		sb.Write(b)
	case nil:
		sb.WriteString("null")
	default:
		fmt.Fprint(sb, t)
	}
}

func writeObject(sb *strings.Builder, dec *json.Decoder, depth int) {
	if !dec.More() {
		dec.Token()
		sb.WriteString("{}")
		return
	}
	sb.WriteString("{\n")
	for dec.More() {
		key, _ := dec.Token()
		sb.WriteString(strings.Repeat("  ", depth+1))
		sb.WriteString(key.(string))
		sb.WriteString(": ")
		writeValue(sb, dec, depth+1)
		sb.WriteString(",\n")
	}
	dec.Token()
	sb.WriteString(strings.Repeat("  ", depth))
	sb.WriteString("}")
}

func writeArray(sb *strings.Builder, dec *json.Decoder, depth int) {
	if !dec.More() {
		dec.Token()
		sb.WriteString("[]")
		return
	}
	sb.WriteString("[\n")
	for dec.More() {
		sb.WriteString(strings.Repeat("  ", depth+1))
		writeValue(sb, dec, depth+1)
		sb.WriteString(",\n")
	}
	dec.Token()
	sb.WriteString(strings.Repeat("  ", depth))
	sb.WriteString("]")
}
