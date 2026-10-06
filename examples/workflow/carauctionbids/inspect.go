package carauctionbids

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// inspect formats a value like Bun's console.log (the runtime that recorded testdata/*.stdout.txt):
// objects span several lines with a trailing comma after each property, arrays of objects put
// the elements on a shared line (`}, {`), strings are double quoted, nil is `undefined`, and
// anything nested deeper than two levels collapses to `[Object ...]` / `[Array ...]`.
// Only the value shapes of this example are supported: structs, maps, slices, strings, numbers,
// booleans and nil.
func inspect(v any) string {
	return inspectAt(reflect.ValueOf(eventAsObject(v)), 0, "")
}

type field struct {
	name  string
	value reflect.Value
}

// eventAsObject returns the JS shape of an engine event: `{ type, ...payload }`.
func eventAsObject(v any) any {
	switch e := v.(type) {
	case xs.InitEvent:
		return []field{{"type", reflect.ValueOf(e.EventType())}, {"input", reflect.ValueOf(e.Input)}}
	case xs.E:
		// Go maps carry no insertion order: `type` first, then the payload keys sorted.
		keys := make([]string, 0, len(e))
		for k := range e {
			if k != "type" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		fields := []field{{"type", reflect.ValueOf(e["type"])}}
		for _, k := range keys {
			fields = append(fields, field{k, reflect.ValueOf(e[k])})
		}
		return fields
	}
	return v
}

func inspectAt(v reflect.Value, depth int, indent string) string {
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer) {
		if v.IsNil() {
			return "undefined"
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return "undefined"
	}
	switch v.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Slice:
		if fs, ok := v.Interface().([]field); ok {
			return inspectFields(fs, depth, indent)
		}
		if v.Len() == 0 {
			return "[]"
		}
		if depth > 2 {
			return "[Array ...]"
		}
		items := make([]string, v.Len())
		for i := range items {
			items[i] = inspectAt(v.Index(i), depth+1, indent+"  ")
		}
		return "[\n" + indent + "  " + strings.Join(items, ", ") + "\n" + indent + "]"
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		fs := make([]field, len(keys))
		for i, k := range keys {
			fs[i] = field{k.String(), v.MapIndex(k)}
		}
		return inspectFields(fs, depth, indent)
	case reflect.Struct:
		var fs []field
		for i := 0; i < v.NumField(); i++ {
			name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("json"), ",")
			fs = append(fs, field{name, v.Field(i)})
		}
		return inspectFields(fs, depth, indent)
	}
	b, _ := json.Marshal(v.Interface())
	return string(b)
}

func inspectFields(fs []field, depth int, indent string) string {
	if depth > 2 {
		return "[Object ...]"
	}
	if len(fs) == 0 {
		return "{}"
	}
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, f := range fs {
		sb.WriteString(indent + "  " + f.name + ": " + inspectAt(f.value, depth+1, indent+"  ") + ",\n")
	}
	sb.WriteString(indent + "}")
	return sb.String()
}
