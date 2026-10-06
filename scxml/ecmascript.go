package scxml

import (
	"reflect"
	"strconv"
	"sync"

	"github.com/dop251/goja"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// ecmascript is the ECMAScript datamodel of one SCXML document. It plays the
// role of the JS host: `eval` for <data expr> and `new Function` for
// executable content (evaluateExecutableContent in scxml.ts).
//
// A goja runtime is not goroutine-safe, and actors of the same machine may
// run on different goroutines, so every evaluation holds mu. Values that
// cross into the machine context are exported to plain Go values
// (map[string]any, []any, numbers, strings, bools, nil, undefinedValue).
// JS functions stay bound to this runtime and are only called back while
// mu is held.
type ecmascript struct {
	mu        sync.Mutex
	vm        *goja.Runtime
	functions map[string]compiled
}

type compiled struct {
	fn  goja.Callable
	err error
}

// undefinedValue distinguishes JS undefined from null, which is Go nil.
type undefinedValue struct{}

func (undefinedValue) MarshalJSON() ([]byte, error) {
	return []byte(`{"xstate$$type":"scxml.undefined"}`), nil
}

func newECMAScript() *ecmascript {
	return &ecmascript{vm: goja.New(), functions: map[string]compiled{}}
}

// evalData mirrors `eval(`(${expr})`)` for a <data> element: expr is
// evaluated in the global scope, without access to other data.
func (e *ecmascript) evalData(expr string) any {
	e.mu.Lock()
	defer e.mu.Unlock()
	v, err := e.vm.RunString("(" + expr + ")")
	if err != nil {
		panic(err)
	}
	return exportJS(v, make(map[*goja.Object]any))
}

// eval mirrors evaluateExecutableContent: body runs inside
// `with (context) { ... }` with `_event = { name, data }` in scope, and its
// return value is exported to Go.
func (e *ecmascript) eval(context map[string]any, event xs.Event, body string) any {
	e.mu.Lock()
	defer e.mu.Unlock()
	return exportJS(e.call(context, event, body), make(map[*goja.Object]any))
}

// test mirrors a guard created by createGuard: the truthiness of
// `return ${cond};`.
func (e *ecmascript) test(context map[string]any, event xs.Event, cond string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.call(context, event, "return "+cond+";").ToBoolean()
}

// call compiles body as JS does with
// `new Function('context', '_event', fnBody)` (lazily, so a syntax error
// surfaces when the content executes) and invokes it. A thrown JS error
// panics, as the JS action or guard throws.
func (e *ecmascript) call(context map[string]any, event xs.Event, body string) goja.Value {
	c, ok := e.functions[body]
	if !ok {
		c = e.compile(body)
		e.functions[body] = c
	}
	if c.err != nil {
		panic(c.err)
	}
	ctx := e.vm.NewObject()
	for k, v := range context {
		if err := ctx.Set(k, e.toJS(v)); err != nil {
			panic(err)
		}
	}
	ev := e.vm.NewObject()
	if err := ev.Set("name", event.EventType()); err != nil {
		panic(err)
	}
	if err := ev.Set("data", e.toJS(eventObject(event))); err != nil {
		panic(err)
	}
	v, err := c.fn(goja.Undefined(), ctx, ev)
	if err != nil {
		panic(err)
	}
	return v
}

func (e *ecmascript) compile(body string) compiled {
	src := "(function (context, _event) {\nconst _sessionid = \"NOT_IMPLEMENTED\";\nwith (context) {\n" + body + "\n}\n})"
	v, err := e.vm.RunString(src)
	if err != nil {
		return compiled{err: err}
	}
	fn, _ := goja.AssertFunction(v)
	return compiled{fn: fn}
}

// exportJS preserves null and undefined before goja's Export collapses both
// to nil. Objects and arrays are traversed recursively for nested values.
func exportJS(v goja.Value, seen map[*goja.Object]any) any {
	if v == nil || goja.IsUndefined(v) {
		return undefinedValue{}
	}
	if goja.IsNull(v) {
		return nil
	}
	if obj, ok := v.(*goja.Object); ok {
		if value, ok := seen[obj]; ok {
			return value
		}
		switch obj.ClassName() {
		case "Array":
			items := make([]any, int(obj.Get("length").ToInteger()))
			seen[obj] = items
			for i := range items {
				items[i] = exportJS(obj.Get(strconv.Itoa(i)), seen)
			}
			return items
		case "Object":
			// Typed arrays and host objects have their own Go representations.
			if obj.ExportType() != reflect.TypeFor[map[string]any]() {
				return obj.Export()
			}
			values := make(map[string]any)
			seen[obj] = values
			for _, key := range obj.Keys() {
				values[key] = exportJS(obj.Get(key), seen)
			}
			return values
		}
	}
	return v.Export()
}

// toJS builds a fresh JS value from a context value, so a script can never
// mutate a map or slice held by a previous snapshot.
func (e *ecmascript) toJS(v any) goja.Value {
	switch t := v.(type) {
	case nil:
		return goja.Null()
	case undefinedValue:
		return goja.Undefined()
	case xs.E:
		return e.toJS(map[string]any(t))
	case map[string]any:
		if len(t) == 1 && t["xstate$$type"] == "scxml.undefined" {
			return goja.Undefined()
		}
		obj := e.vm.NewObject()
		for k, item := range t {
			if err := obj.Set(k, e.toJS(item)); err != nil {
				panic(err)
			}
		}
		return obj
	case []any:
		items := make([]any, len(t))
		for i, item := range t {
			items[i] = e.toJS(item)
		}
		return e.vm.NewArray(items...)
	}
	return e.vm.ToValue(v)
}

// eventObject mirrors the plain JS event object exposed as `_event.data`.
func eventObject(event xs.Event) any {
	switch ev := event.(type) {
	case xs.E:
		return map[string]any(ev)
	case xs.InitEvent:
		return map[string]any{"type": ev.EventType(), "input": ev.Input}
	case xs.DoneStateEvent:
		return map[string]any{"type": ev.EventType(), "output": ev.Output}
	case xs.DoneActorEvent:
		return map[string]any{"type": ev.EventType(), "output": ev.Output, "actorId": ev.ActorID}
	case xs.ErrorActorEvent:
		return map[string]any{"type": ev.EventType(), "error": ev.Error, "actorId": ev.ActorID}
	}
	return map[string]any{"type": event.EventType()}
}
