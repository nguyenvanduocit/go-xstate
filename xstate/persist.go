package xstate

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Port of State.ts persistContext and StateMachine.ts reviveContext. JS
// replaces actor refs inside the context with `{ xstate$$type: 1, id }`
// markers when persisting and swaps them back for the restored children.

const actorTypeMarker = 1

// actorMarker stands for a persisted actor ref in a context slot whose static
// type is an interface other than `any` (typically ActorRef).
type actorMarker struct{ id string }

func (m *actorMarker) ID() string            { return m.id }
func (m *actorMarker) SessionID() string     { return "" }
func (m *actorMarker) Send(Event)            {}
func (m *actorMarker) AnySnapshot() Snapshot { return nil }
func (m *actorMarker) SubscribeAny(Observer[Snapshot]) Subscription {
	return SubscriptionFunc(func() {})
}
func (m *actorMarker) System() *ActorSystem         { return nil }
func (m *actorMarker) Src() any                     { return nil }
func (m *actorMarker) Parent() ActorRef             { return nil }
func (m *actorMarker) MarshalJSON() ([]byte, error) { return json.Marshal(markerMap(m.id)) }

func markerMap(id string) map[string]any {
	return map[string]any{"xstate$$type": actorTypeMarker, "id": id}
}

var (
	actorRefType = reflect.TypeFor[ActorRef]()
	anyType      = reflect.TypeFor[any]()
)

func isLiveActorRef(v reflect.Value) (ActorRef, bool) {
	if !v.IsValid() || !v.CanInterface() {
		return nil, false
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
	}
	ref, ok := v.Interface().(ActorRef)
	if !ok || isNilRef(ref) {
		return nil, false
	}
	if _, isMarker := ref.(*actorMarker); isMarker {
		return nil, false
	}
	return ref, true
}

// persistContext mirrors State.ts persistContext.
func persistContext(ctx any) any {
	if ctx == nil {
		return nil
	}
	v := reflect.ValueOf(ctx)
	out, changed := persistValue(v, v.Type())
	if !changed {
		return ctx
	}
	return out.Interface()
}

// persistValue replaces actor refs without mutating the live context.
func persistValue(v reflect.Value, slot reflect.Type) (reflect.Value, bool) {
	return transformContext(v, slot, func(v reflect.Value, slot reflect.Type) (reflect.Value, bool, bool) {
		if ref, ok := isLiveActorRef(v); ok {
			if slot == anyType {
				return reflect.ValueOf(markerMap(ref.ID())), true, true
			}
			if slot.Kind() == reflect.Interface && reflect.TypeFor[*actorMarker]().Implements(slot) {
				out := reflect.New(slot).Elem()
				out.Set(reflect.ValueOf(&actorMarker{id: ref.ID()}))
				return out, true, true
			}
			// A concrete actor pointer cannot hold a marker. Do not copy its internals.
			return v, true, false
		}
		return v, false, false
	})
}

// contextVisit identifies graph nodes, including distinct views of a slice.
type contextVisit struct {
	typ      reflect.Type
	ptr      uintptr
	len, cap int
}

// contextCopy records incoming edges so replacement requirements propagate
// through cycles before any decisions about retaining original pointers.
type contextCopy struct {
	parents []*contextCopy
	changed bool
}

func contextIdentity(v reflect.Value) (contextVisit, bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if !v.IsNil() {
			key := contextVisit{typ: v.Type(), ptr: v.Pointer()}
			if v.Kind() == reflect.Slice {
				key.len, key.cap = v.Len(), v.Cap()
			}
			return key, true
		}
	}
	return contextVisit{}, false
}

// transformContext discovers the graph, marks ancestors of replacements, then
// copies only marked nodes. Unchanged resources keep their identity; changed
// cycles and repeated references resolve to the same copy.
func transformContext(v reflect.Value, slot reflect.Type, leaf func(reflect.Value, reflect.Type) (reflect.Value, bool, bool)) (reflect.Value, bool) {
	root := &contextCopy{}
	nodes := map[contextVisit]*contextCopy{}
	var changed []*contextCopy
	var discover func(reflect.Value, reflect.Type, *contextCopy)
	discover = func(v reflect.Value, slot reflect.Type, parent *contextCopy) {
		if !v.IsValid() {
			return
		}
		if _, handled, replaced := leaf(v, slot); handled {
			if replaced && !parent.changed {
				parent.changed = true
				changed = append(changed, parent)
			}
			return
		}
		if key, ok := contextIdentity(v); ok {
			if node, exists := nodes[key]; exists {
				node.parents = append(node.parents, parent)
				return
			}
			node := &contextCopy{parents: []*contextCopy{parent}}
			nodes[key] = node
			parent = node
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if !v.IsNil() {
				discover(v.Elem(), v.Elem().Type(), parent)
			}
		case reflect.Map:
			iter := v.MapRange()
			for iter.Next() {
				discover(iter.Value(), v.Type().Elem(), parent)
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				discover(v.Index(i), v.Type().Elem(), parent)
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if f := v.Type().Field(i); f.IsExported() {
					discover(v.Field(i), f.Type, parent)
				}
			}
		}
	}
	discover(v, slot, root)
	for i := 0; i < len(changed); i++ {
		for _, parent := range changed[i].parents {
			if !parent.changed {
				parent.changed = true
				changed = append(changed, parent)
			}
		}
	}
	if !root.changed {
		return v, false
	}
	seen := map[contextVisit]reflect.Value{}
	var visit func(reflect.Value, reflect.Type) reflect.Value
	visit = func(v reflect.Value, slot reflect.Type) reflect.Value {
		if !v.IsValid() {
			return v
		}
		if out, handled, _ := leaf(v, slot); handled {
			return out
		}
		key, identified := contextIdentity(v)
		if identified {
			if !nodes[key].changed {
				return v
			}
			if out, ok := seen[key]; ok {
				return out
			}
		} else {
			switch v.Kind() {
			case reflect.Pointer, reflect.Map, reflect.Slice:
				return v // nil
			}
		}
		switch v.Kind() {
		case reflect.Interface:
			if v.IsNil() {
				return v
			}
			out := reflect.New(v.Type()).Elem()
			out.Set(visit(v.Elem(), v.Elem().Type()))
			return out
		case reflect.Pointer:
			out := reflect.New(v.Type().Elem())
			if out.Type() != v.Type() {
				out = out.Convert(v.Type())
			}
			seen[key] = out
			out.Elem().Set(visit(v.Elem(), v.Type().Elem()))
			return out
		case reflect.Map:
			out := reflect.MakeMapWithSize(v.Type(), v.Len())
			seen[key] = out
			iter := v.MapRange()
			for iter.Next() {
				out.SetMapIndex(iter.Key(), visit(iter.Value(), v.Type().Elem()))
			}
			return out
		case reflect.Slice:
			out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
			seen[key] = out
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(visit(v.Index(i), v.Type().Elem()))
			}
			return out
		case reflect.Array:
			out := reflect.New(v.Type()).Elem()
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(visit(v.Index(i), v.Type().Elem()))
			}
			return out
		case reflect.Struct:
			out := reflect.New(v.Type()).Elem()
			out.Set(v)
			for i := 0; i < v.NumField(); i++ {
				if f := v.Type().Field(i); f.IsExported() {
					out.Field(i).Set(visit(v.Field(i), f.Type))
				}
			}
			return out
		}
		return v
	}
	return visit(v, slot), true
}

// restoreContext converts a persisted context to C and revives actor refs.
func restoreContext[C any](v any, children map[string]ActorRef) any {
	t := reflect.TypeFor[C]()
	if v == nil {
		return nil
	}
	if c, ok := v.(C); ok && t.Kind() != reflect.Interface {
		out, changed := reviveValue(reflect.ValueOf(c), t, children)
		if changed {
			return out.Interface()
		}
		return c
	}
	dst := reflect.New(t).Elem()
	convertInto(dst, v, children)
	return dst.Interface()
}

func markerID(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	switch tv := m["xstate$$type"].(type) {
	case int:
		if tv != actorTypeMarker {
			return "", false
		}
	case float64:
		if tv != actorTypeMarker {
			return "", false
		}
	default:
		return "", false
	}
	id, ok := m["id"].(string)
	return id, ok
}

// reviveValue replaces markers in v (of static type slot) with children.
func reviveValue(v reflect.Value, slot reflect.Type, children map[string]ActorRef) (reflect.Value, bool) {
	return transformContext(v, slot, func(v reflect.Value, slot reflect.Type) (reflect.Value, bool, bool) {
		if v.CanInterface() {
			id, marker := markerID(v.Interface())
			if m, ok := v.Interface().(*actorMarker); ok && m != nil {
				id, marker = m.id, true
			}
			if marker {
				out := reflect.New(slot).Elem()
				if ref := children[id]; ref != nil && reflect.TypeOf(ref).AssignableTo(slot) {
					out.Set(reflect.ValueOf(ref))
				}
				return out, true, true
			}
			if _, ok := v.Interface().(ActorRef); ok {
				return v, true, false
			}
		}
		return v, false, false
	})
}

// convertInto stores src (a JSON-like value) into dst, converting types the
// way encoding/json would and reviving actor markers.
func convertInto(dst reflect.Value, src any, children map[string]ActorRef) {
	if src == nil {
		return
	}
	if id, ok := markerID(src); ok {
		if ref := children[id]; ref != nil && reflect.TypeOf(ref).AssignableTo(dst.Type()) {
			dst.Set(reflect.ValueOf(ref))
		}
		return
	}
	sv := reflect.ValueOf(src)
	if dst.Kind() == reflect.Interface {
		if sv.Type().AssignableTo(dst.Type()) {
			out, _ := reviveValue(sv, sv.Type(), children)
			dst.Set(out)
		}
		return
	}
	if sv.Type().AssignableTo(dst.Type()) {
		out, _ := reviveValue(sv, dst.Type(), children)
		dst.Set(out)
		return
	}
	switch dst.Kind() {
	case reflect.Pointer:
		p := reflect.New(dst.Type().Elem())
		convertInto(p.Elem(), src, children)
		dst.Set(p)
		return
	case reflect.Struct:
		m, ok := src.(map[string]any)
		if !ok {
			break
		}
		for i := 0; i < dst.NumField(); i++ {
			f := dst.Type().Field(i)
			if !f.IsExported() {
				continue
			}
			name := f.Name
			if tag := f.Tag.Get("json"); tag != "" {
				if n := strings.Split(tag, ",")[0]; n == "-" {
					continue
				} else if n != "" {
					name = n
				}
			}
			for k, val := range m {
				if strings.EqualFold(k, name) {
					convertInto(dst.Field(i), val, children)
					break
				}
			}
		}
		return
	case reflect.Map:
		m, ok := src.(map[string]any)
		if !ok || dst.Type().Key().Kind() != reflect.String {
			break
		}
		out := reflect.MakeMapWithSize(dst.Type(), len(m))
		for k, val := range m {
			ev := reflect.New(dst.Type().Elem()).Elem()
			convertInto(ev, val, children)
			out.SetMapIndex(reflect.ValueOf(k).Convert(dst.Type().Key()), ev)
		}
		dst.Set(out)
		return
	case reflect.Slice:
		arr, ok := src.([]any)
		if !ok {
			break
		}
		out := reflect.MakeSlice(dst.Type(), len(arr), len(arr))
		for i, val := range arr {
			convertInto(out.Index(i), val, children)
		}
		dst.Set(out)
		return
	}
	if sv.Type().ConvertibleTo(dst.Type()) {
		switch dst.Kind() {
		case reflect.String:
			if sv.Kind() != reflect.String {
				return
			}
		}
		dst.Set(sv.Convert(dst.Type()))
		return
	}
	b, err := json.Marshal(src)
	if err == nil {
		p := reflect.New(dst.Type())
		if json.Unmarshal(b, p.Interface()) == nil {
			dst.Set(p.Elem())
		}
	}
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
