package store_test

import (
	"fmt"
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// storeTypes1LogSchema mirrors
//
//	z.discriminatedUnion('level', [
//	  z.object({ level: z.literal('warn'), message: z.string() }),
//	  z.object({ level: z.literal('error'), error: z.string() })
//	])
//
// A value is valid when it satisfies one of the object variants.
func storeTypes1LogSchema() xstore.Schema {
	variants := []xstore.Schema{
		zObject(map[string]*zSchema{"level": zLiteral("warn"), "message": zString()}),
		zObject(map[string]*zSchema{"level": zLiteral("error"), "error": zString()}),
	}
	return xstore.SchemaFunc(func(v any) (xstore.SchemaResult, *xstore.Promise[xstore.SchemaResult]) {
		var issues []xstore.SchemaIssue
		for _, variant := range variants {
			res, _ := variant.Validate(v)
			if len(res.Issues) == 0 {
				return xstore.SchemaResult{Value: v}, nil
			}
			issues = append(issues, res.Issues...)
		}
		return xstore.SchemaResult{Issues: issues}, nil
	})
}

// storeTypes1Int reads an int payload field; a missing or mistyped field
// yields 0 (JS reads `undefined` / a string without throwing).
func storeTypes1Int(ev xs.Event, key string) int {
	n, _ := ev.(xs.E)[key].(int)
	return n
}

// ---- emitted ----

// JS: emitted > can emit a known event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L11
func TestStoreTypes_Emitted_CanEmitAKnownEvent(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return c, true
			},
		},
	})

	assert.NotNil(t, s)
}

// JS: emitted > can't emit an unknown event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L28
func TestStoreTypes_Emitted_CantEmitAnUnknownEvent(t *testing.T) {
	type ctx struct{}

	// The JS handler is never invoked; `.unknown()` only fails type checking.
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
				"decreased": zObject(map[string]*zSchema{"downBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("unknown")
				return c, true
			},
		},
	})

	assert.NotNil(t, s)
}

// JS: emitted > can't emit a known event with wrong payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L48
func TestStoreTypes_Emitted_CantEmitAKnownEventWithWrongPayload(t *testing.T) {
	type ctx struct{}

	// The JS handler is never invoked; the wrong payload only fails type checking.
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
				"decreased": zObject(map[string]*zSchema{"downBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("increased", xs.E{"upBy": "bazinga"})
				return c, true
			},
		},
	})

	assert.NotNil(t, s)
}

// JS: emitted > can subscribe to a known event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L69
func TestStoreTypes_Emitted_CanSubscribeToAKnownEvent(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	assert.NotPanics(t, func() {
		s.On("increased", func(ev xs.Event) {
			// JS: ev satisfies { type: 'increased'; upBy: number } (type-only)
			_ = ev
		})
	})
}

// JS: emitted > can't subscribe to a unknown event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L87
func TestStoreTypes_Emitted_CantSubscribeToAUnknownEvent(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		Context: ctx{},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	assert.NotPanics(t, func() {
		s.On("increased", func(ev xs.Event) {})
		// JS: @ts-expect-error — the unknown type is rejected by the compiler only.
		s.On("unknown", func(ev xs.Event) {})
	})
}

// JS: emitted > wildcard listener receives union of all emitted events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L107
func TestStoreTypes_Emitted_WildcardListenerReceivesUnionOfAllEmittedEvents(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
				"decreased": zObject(map[string]*zSchema{"downBy": zNumber()}),
			},
		},
		Context: ctx{},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	assert.NotPanics(t, func() {
		s.On("*", func(ev xs.Event) {
			// JS: ev satisfies the union of emitted events (type-only)
			_ = ev
		})
	})
}

// JS: emitted > works with a discriminated union event payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L129
func TestStoreTypes_Emitted_WorksWithADiscriminatedUnionEventPayload(t *testing.T) {
	type ctx struct{}

	// The JS handler is never invoked; the mismatching third payload only
	// fails type checking.
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{"log": storeTypes1LogSchema()},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"log": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("log", xs.E{"level": "warn", "message": "hmm"})
				enq.Emit("log", xs.E{"level": "error", "error": "uh oh"})
				enq.Emit("log", xs.E{"level": "error", "message": "foo"})
				return c, true
			},
		},
	})

	assert.NotNil(t, s)
}

// ---- trigger ----

// JS: trigger > works with a distributive event payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L157
func TestStoreTypes_Trigger_WorksWithADistributiveEventPayload(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		On: map[string]xstore.StoreAssigner[ctx]{
			"log": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return c, true
			},
		},
	})

	assert.NotPanics(t, func() {
		s.Trigger("log", xs.E{"level": "warn", "message": "hmm"})
		s.Trigger("log", xs.E{"level": "error", "error": "uh oh"})

		// JS: @ts-expect-error — still executes at runtime.
		s.Trigger("log", xs.E{"level": "error", "message": "foo"})
	})
}

// JS: trigger > uses schema-declared events for trigger typing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L182
func TestStoreTypes_Trigger_UsesSchemaDeclaredEventsForTriggerTyping(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{"log": storeTypes1LogSchema()},
		},
		Context: ctx{},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	assert.NotPanics(t, func() {
		s.Trigger("log", xs.E{"level": "warn", "message": "hmm"})
		s.Trigger("log", xs.E{"level": "error", "error": "uh oh"})

		// JS: @ts-expect-error — still executes at runtime.
		s.Trigger("log", xs.E{"level": "error", "message": "foo"})
	})
}

// JS: trigger > preserves inferred trigger typing when only emitted schemas are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L206
func TestStoreTypes_Trigger_PreservesInferredTriggerTypingWhenOnlyEmittedSchemasAreDeclared(t *testing.T) {
	type ctx struct{}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"logged": zObject(map[string]*zSchema{"message": zString()}),
			},
		},
		Context: ctx{},
		On: map[string]xstore.StoreAssigner[ctx]{
			"log": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("logged", xs.E{"message": ev.(xs.E)["message"]})
				return c, true
			},
		},
	})

	assert.NotPanics(t, func() {
		s.Trigger("log", xs.E{"message": "hello"})
	})

	// JS: the `if (false) { ... }` block holds only @ts-expect-error checks
	// (`trigger.log({})`, `trigger.unknown()`) and never runs.
}

// JS: trigger > uses schema-declared events for enqueued trigger typing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L233
func TestStoreTypes_Trigger_UsesSchemaDeclaredEventsForEnqueuedTriggerTyping(t *testing.T) {
	type ctx struct{}

	// The JS handler is never invoked; the bad calls only fail type checking.
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"log":   storeTypes1LogSchema(),
				"flush": zObject(map[string]*zSchema{}),
			},
		},
		Context: ctx{},
		On: map[string]xstore.StoreAssigner[ctx]{
			"flush": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Trigger("flush")
				enq.Trigger("flush", xs.E{})
				enq.Trigger("log", xs.E{"level": "warn", "message": "hmm"})
				enq.Trigger("log", xs.E{"level": "error", "error": "uh oh"})

				enq.Trigger("log", xs.E{"level": "error", "message": "foo"})

				enq.Trigger("unknown")

				return c, true
			},
		},
	})

	assert.NotNil(t, s)
}

// ---- can ----

// JS: can > uses event payload types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L269
func TestStoreTypes_Can_UsesEventPayloadTypes(t *testing.T) {
	type ctx struct{ Count int }

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"increment": zObject(map[string]*zSchema{"by": zNumber()}),
				"reset":     zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"increment": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + storeTypes1Int(ev, "by")}, true
			},
			"reset": func(ctx, xs.Event, *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: 0}, true
			},
		},
	})

	assert.NotPanics(t, func() {
		_ = s.Can("increment", xs.E{"by": 1})
		_ = s.Can("reset")
		_ = s.Can("reset", xs.E{})

		// JS: @ts-expect-error (missing payload) — still executes at runtime.
		_ = s.Can("increment")
		// JS: @ts-expect-error (by: 'one') — still executes at runtime.
		_ = s.Can("increment", xs.E{"by": "one"})
	})
}

// ---- logic selectors ----

// JS: logic selectors > infers selected values from a store
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L298
func TestStoreTypes_LogicSelectors_InfersSelectedValuesFromAStore(t *testing.T) {
	type ctx struct{ Count int }

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	})
	count := xstore.Select(s, func(c ctx) int { return c.Count })
	label := xstore.Select(s, func(c ctx) string { return fmt.Sprintf("Count: %d", c.Count) })

	// JS: `satisfies number` / `satisfies string` are type-only; the runtime
	// values are asserted here.
	assert.Equal(t, 0, count.Get())
	assert.Equal(t, "Count: 0", label.Get())

	// JS: `if (false) { count.get() satisfies string }` never runs (type-only).
}

// JS: logic selectors > infers input and selector values from reusable store logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L317
func TestStoreTypes_LogicSelectors_InfersInputAndSelectorValuesFromReusableStoreLogic(t *testing.T) {
	type input struct{ InitialCount int }
	type ctx struct{ Count int }

	counterLogic := xstore.CreateStoreLogic(xstore.StoreConfig[ctx]{
		ContextFn: func(in any) ctx {
			return ctx{Count: in.(input).InitialCount}
		},
		Selectors: map[string]func(ctx) any{
			"count": func(c ctx) any { return c.Count },
			"label": func(c ctx) any { return fmt.Sprintf("Count: %d", c.Count) },
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	})

	s := counterLogic.CreateStore(input{InitialCount: 1})

	// JS: `satisfies number` / `satisfies string` are type-only; the runtime
	// values are asserted here.
	assert.Equal(t, 1, s.Selectors()["count"].Get())
	assert.Equal(t, "Count: 1", s.Selectors()["label"].Get())

	// JS: the `if (false) { ... }` block holds only @ts-expect-error checks
	// (missing / undefined / mistyped createStore input, wrong selector type)
	// and never runs.
}

// ---- schemas ----

// JS: schemas > requires event and emitted schemas to define object payloads
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L355
func TestStoreTypes_Schemas_RequiresEventAndEmittedSchemasToDefineObjectPayloads(t *testing.T) {
	t.Skip("N/A: type-level only — both createStore calls are @ts-expect-error (a z.string() schema is not an object-payload schema); Go's Schema interface has no payload-shape constraint")
}

// JS: schemas > uses schema-declared context for snapshot typing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L379
func TestStoreTypes_Schemas_UsesSchemaDeclaredContextForSnapshotTyping(t *testing.T) {
	type ctx struct {
		Count int    `json:"count"`
		Label string `json:"label"`
	}

	schemas := &xstore.StoreSchemas{
		Context: zObject(map[string]*zSchema{"count": zNumber(), "label": zString()}),
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: schemas,
		Context: ctx{Count: 0, Label: "ready"},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	// JS: `satisfies string` / `satisfies StoreSchemas | undefined` are
	// type-only; the runtime values are asserted here.
	assert.Equal(t, "ready", s.GetSnapshot().Context.Label)
	assert.Same(t, schemas, s.Schemas())

	// JS: `getSnapshot().context.label satisfies number` is @ts-expect-error (type-only).
}

// JS: schemas > merges schema-declared context with inferred event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L399
func TestStoreTypes_Schemas_MergesSchemaDeclaredContextWithInferredEventTypes(t *testing.T) {
	type ctx struct {
		Count int    `json:"count"`
		Label string `json:"label"`
	}

	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Context: zObject(map[string]*zSchema{"count": zNumber(), "label": zString()}),
		},
		Context: ctx{Count: 0, Label: "ready"},
		On: map[string]xstore.StoreAssigner[ctx]{
			"rename": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				next := c
				next.Label = ev.(xs.E)["label"].(string)
				return next, true
			},
		},
	})

	s.Trigger("rename", xs.E{"label": "done"})
	// JS: `satisfies string` is type-only; the runtime value is asserted here.
	assert.Equal(t, "done", s.GetSnapshot().Context.Label)

	// JS: `if (false) { store.trigger.rename({}) }` is @ts-expect-error and never runs.
}

// ---- fromStore schemas ----

// JS: fromStore schemas > preserves inferred event types when only emitted schemas are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L427
func TestStoreTypes_FromStoreSchemas_PreservesInferredEventTypesWhenOnlyEmittedSchemasAreDeclared(t *testing.T) {
	type ctx struct{ Count int }

	logic := xstore.FromStore(xstore.StoreConfig[ctx]{
		ContextFn: func(in any) ctx { return ctx{Count: in.(int)} },
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				by := storeTypes1Int(ev, "by")
				enq.Emit("increased", xs.E{"upBy": by})
				return ctx{Count: c.Count + by}, true
			},
		},
	})

	actor := xs.CreateActor(logic, xs.WithInput(1))

	assert.NotPanics(t, func() {
		actor.Send(xs.E{"type": "inc", "by": 2})
		actor.On("increased", func(event xs.Event) {
			// JS: event.upBy satisfies number (type-only)
			_ = event
		})
	})

	// JS: the `if (false) { ... }` block holds only @ts-expect-error checks
	// (`send({ message })`, `on('unknown')`) and never runs.
}

// JS: fromStore schemas > uses schema-declared events for send typing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L469
func TestStoreTypes_FromStoreSchemas_UsesSchemaDeclaredEventsForSendTyping(t *testing.T) {
	type ctx struct{ Count int }

	logic := xstore.FromStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc":   zObject(map[string]*zSchema{"by": zNumber()}),
				"reset": zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + storeTypes1Int(ev, "by")}, true
			},
		},
	})

	actor := xs.CreateActor(logic)

	assert.NotPanics(t, func() {
		actor.Send(xs.E{"type": "inc", "by": 1})
		actor.Send(xs.E{"type": "reset"})
	})

	// JS: the `if (false) { ... }` block holds only @ts-expect-error checks
	// (`send({ type: 'inc' })`, `send({ type: 'unknown' })`) and never runs.
}

// JS: fromStore schemas > uses schema-declared context for snapshot typing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/types.test.tsx#L501
func TestStoreTypes_FromStoreSchemas_UsesSchemaDeclaredContextForSnapshotTyping(t *testing.T) {
	type ctx struct {
		Count int    `json:"count"`
		Label string `json:"label"`
	}

	logic := xstore.FromStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Context: zObject(map[string]*zSchema{"count": zNumber(), "label": zString()}),
		},
		Context: ctx{Count: 0, Label: "ready"},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	snapshot := logic.GetInitialSnapshot(&xs.ActorScope{}, nil)

	// JS: `satisfies string` is type-only; the runtime value is asserted here.
	assert.Equal(t, "ready", snapshot.Context.Label)

	// JS: `snapshot.context.label satisfies number` is @ts-expect-error (type-only).
}
