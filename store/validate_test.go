package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storeValidate1Ctx is a context whose count is deliberately untyped so tests
// can hold invalid values such as "nope" (JS: `as any`).
type storeValidate1Ctx struct {
	Count any `json:"count"`
}

// storeValidate1IntCtx is the plain `{ count: number }` context.
type storeValidate1IntCtx struct {
	Count int `json:"count"`
}

// storeValidate1Num converts a numeric event payload value to int.
func storeValidate1Num(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	panic("storeValidate1Num: not a number")
}

// storeValidate1Thrown mirrors getThrown(fn) in validate.test.ts and requires
// that the thrown value is a *StoreValidationError.
func storeValidate1Thrown(t *testing.T, fn func()) *xstore.StoreValidationError {
	t.Helper()
	ve, ok := recovered(fn).(*xstore.StoreValidationError)
	require.True(t, ok, "expected fn to panic with *StoreValidationError")
	return ve
}

// storeValidate1ToThrow mirrors expect(fn).toThrow(StoreValidationError).
func storeValidate1ToThrow(t *testing.T, fn func()) {
	t.Helper()
	err, _ := recovered(fn).(error)
	var ve *xstore.StoreValidationError
	assert.ErrorAs(t, err, &ve)
}

// JS: validates initial context when the extension is applied
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L14
func TestStoreValidate_ValidatesInitialContextWhenTheExtensionIsApplied(t *testing.T) {
	storeValidate1ToThrow(t, func() {
		xstore.CreateStore(xstore.StoreConfig[storeValidate1Ctx]{
			Schemas: &xstore.StoreSchemas{
				Context: zObject(map[string]*zSchema{"count": zNumber()}),
			},
			Context: storeValidate1Ctx{Count: "nope"},
			On:      map[string]xstore.StoreAssigner[storeValidate1Ctx]{},
		}).With(xstore.ValidateSchemas[storeValidate1Ctx]())
	})
}

// JS: validates event payloads before transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L26
func TestStoreValidate_ValidatesEventPayloadsBeforeTransitions(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc": zObject(map[string]*zSchema{"by": zNumber()}),
			},
		},
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + storeValidate1Num(ev.(xs.E)["by"])}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx]())

	assert.False(t, s.Can("inc", xs.E{"by": "nope"}))

	storeValidate1ToThrow(t, func() { s.Trigger("inc", xs.E{"by": "nope"}) })
	assert.Equal(t, storeValidate1IntCtx{Count: 0}, s.GetSnapshot().Context)

	s.Trigger("inc", xs.E{"by": 2})
	assert.Equal(t, storeValidate1IntCtx{Count: 2}, s.GetSnapshot().Context)
}

// JS: validates final context after a macrostep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L50
func TestStoreValidate_ValidatesFinalContextAfterAMacrostep(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1Ctx]{
		Schemas: &xstore.StoreSchemas{
			Events:  map[string]xstore.Schema{"break": zObject(map[string]*zSchema{})},
			Context: zObject(map[string]*zSchema{"count": zNumber()}),
		},
		Context: storeValidate1Ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1Ctx]{
			"break": func(c storeValidate1Ctx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1Ctx]) (storeValidate1Ctx, bool) {
				return storeValidate1Ctx{Count: "nope"}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1Ctx]())

	assert.False(t, s.Can("break"))
	storeValidate1ToThrow(t, func() { s.Trigger("break") })
}

// JS: validates emitted payloads before running effects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L68
func TestStoreValidate_ValidatesEmittedPayloadsBeforeRunningEffects(t *testing.T) {
	effectSpy := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{"send": zObject(map[string]*zSchema{})},
			Emitted: map[string]xstore.Schema{
				"sent": zObject(map[string]*zSchema{"value": zNumber()}),
			},
		},
		Context: struct{}{},
		On: map[string]xstore.StoreAssigner[struct{}]{
			"send": func(c struct{}, ev xs.Event, enq *xstore.EnqueueObject[struct{}]) (struct{}, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[struct{}]) { effectSpy.Call() })
				enq.Emit("sent", xs.E{"value": "nope"})
				return c, true
			},
		},
	}).With(xstore.ValidateSchemas[struct{}]())

	storeValidate1ToThrow(t, func() { s.Trigger("send") })
	assert.Equal(t, 0, effectSpy.Count())
}

// JS: validates no-payload events and emitted events as empty objects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L93
func TestStoreValidate_ValidatesNoPayloadEventsAndEmittedEventsAsEmptyObjects(t *testing.T) {
	emittedSpy := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Events:  map[string]xstore.Schema{"reset": zObject(map[string]*zSchema{})},
			Emitted: map[string]xstore.Schema{"reset": zObject(map[string]*zSchema{})},
		},
		Context: storeValidate1IntCtx{Count: 1},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"reset": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				enq.Emit("reset")
				return storeValidate1IntCtx{Count: 0}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx]())

	s.On("reset", func(e xs.Event) { emittedSpy.Call(e) })
	s.Trigger("reset")

	assert.Equal(t, storeValidate1IntCtx{Count: 0}, s.GetSnapshot().Context)
	assert.Contains(t, emittedSpy.Calls(), []any{xs.E{"type": "reset"}})
}

// JS: throws for unknown events and emitted events by default
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L120
func TestStoreValidate_ThrowsForUnknownEventsAndEmittedEventsByDefault(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events:  map[string]xstore.Schema{"send": zObject(map[string]*zSchema{})},
			Emitted: map[string]xstore.Schema{"known": zObject(map[string]*zSchema{})},
		},
		Context: struct{}{},
		On: map[string]xstore.StoreAssigner[struct{}]{
			"send": func(c struct{}, ev xs.Event, enq *xstore.EnqueueObject[struct{}]) (struct{}, bool) {
				enq.Emit("unknown")
				return c, true
			},
		},
	}).With(xstore.ValidateSchemas[struct{}]())

	ve := storeValidate1Thrown(t, func() { s.Send(xs.E{"type": "unknown"}) })
	assert.Equal(t, xstore.ReasonUnknownEvent, ve.Reason)
	assert.Equal(t, "unknown", ve.EventType)
	assert.JSONEq(t, `{}`, toJSON(t, ve.Payload))

	ve = storeValidate1Thrown(t, func() { s.Trigger("send") })
	assert.Equal(t, xstore.ReasonUnknownEmitted, ve.Reason)
	assert.Equal(t, "unknown", ve.EventType)
	assert.JSONEq(t, `{}`, toJSON(t, ve.Payload))
}

// JS: returns false from can for validation errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L153
func TestStoreValidate_ReturnsFalseFromCanForValidationErrors(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc": zObject(map[string]*zSchema{"by": zNumber()}),
			},
		},
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + storeValidate1Num(ev.(xs.E)["by"])}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx]())

	assert.False(t, s.Can("inc", xs.E{"by": "nope"}))
}

// JS: exposes validation error details
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L169
func TestStoreValidate_ExposesValidationErrorDetails(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc": zObject(map[string]*zSchema{"by": zNumber()}),
			},
		},
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + storeValidate1Num(ev.(xs.E)["by"])}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx]())

	thrown := recovered(func() { s.Trigger("inc", xs.E{"by": "nope"}) })
	// name: 'StoreValidationError'
	assert.True(t, xstore.IsStoreValidationError(thrown))
	ve, ok := thrown.(*xstore.StoreValidationError)
	require.True(t, ok)
	assert.Equal(t, xstore.ReasonInvalidEvent, ve.Reason)
	assert.Equal(t, "inc", ve.EventType)
	assert.JSONEq(t, `{"by":"nope"}`, toJSON(t, ve.Payload))
	assert.NotEmpty(t, ve.Issues) // issues: expect.any(Array)
}

// JS: throws for unknown emitted events by default
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L193
func TestStoreValidate_ThrowsForUnknownEmittedEventsByDefault(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events:  map[string]xstore.Schema{"send": zObject(map[string]*zSchema{})},
			Emitted: map[string]xstore.Schema{"known": zObject(map[string]*zSchema{})},
		},
		Context: struct{}{},
		On: map[string]xstore.StoreAssigner[struct{}]{
			"send": func(c struct{}, ev xs.Event, enq *xstore.EnqueueObject[struct{}]) (struct{}, bool) {
				enq.Emit("unknown")
				return c, true
			},
		},
	}).With(xstore.ValidateSchemas[struct{}]())

	ve := storeValidate1Thrown(t, func() { s.Trigger("send") })
	assert.Equal(t, xstore.ReasonUnknownEmitted, ve.Reason)
	assert.Equal(t, "unknown", ve.EventType)
	assert.JSONEq(t, `{}`, toJSON(t, ve.Payload))
}

// JS: throws for unknown events by default
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L219
func TestStoreValidate_ThrowsForUnknownEventsByDefault(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{"send": zObject(map[string]*zSchema{})},
		},
		Context: struct{}{},
		On:      map[string]xstore.StoreAssigner[struct{}]{},
	}).With(xstore.ValidateSchemas[struct{}]())

	ve := storeValidate1Thrown(t, func() { s.Send(xs.E{"type": "unknown"}) })
	assert.Equal(t, xstore.ReasonUnknownEvent, ve.Reason)
	assert.Equal(t, "unknown", ve.EventType)
	assert.JSONEq(t, `{}`, toJSON(t, ve.Payload))
}

// JS: can ignore unknown events and emitted events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L239
func TestStoreValidate_CanIgnoreUnknownEventsAndEmittedEvents(t *testing.T) {
	emittedSpy := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events:  map[string]xstore.Schema{"send": zObject(map[string]*zSchema{})},
			Emitted: map[string]xstore.Schema{"known": zObject(map[string]*zSchema{})},
		},
		Context: struct{}{},
		On: map[string]xstore.StoreAssigner[struct{}]{
			"send": func(c struct{}, ev xs.Event, enq *xstore.EnqueueObject[struct{}]) (struct{}, bool) {
				enq.Emit("unknown")
				return c, true
			},
		},
	}).With(xstore.ValidateSchemas[struct{}](xstore.ValidateSchemasOptions{
		UnknownEvents:  xstore.UnknownIgnore,
		UnknownEmitted: xstore.UnknownIgnore,
	}))

	s.On("*", func(e xs.Event) { emittedSpy.Call(e) })
	s.Send(xs.E{"type": "unknown"})
	s.Trigger("send")

	assert.Contains(t, emittedSpy.Calls(), []any{xs.E{"type": "unknown"}})
}

// JS: allows extension-added event types without schemas
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L271
func TestStoreValidate_AllowsExtensionAddedEventTypesWithoutSchemas(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Context: zObject(map[string]*zSchema{"count": zNumber()}),
			Events:  map[string]xstore.Schema{"inc": zObject(map[string]*zSchema{})},
		},
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + 1}, true
			},
		},
	}).
		With(xstore.Reset[storeValidate1IntCtx]()).
		With(xstore.ValidateSchemas[storeValidate1IntCtx]())

	s.Trigger("inc")
	s.Trigger("reset")

	assert.Equal(t, storeValidate1IntCtx{Count: 0}, s.GetSnapshot().Context)
}

// JS: can opt out of individual validation areas
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L293
func TestStoreValidate_CanOptOutOfIndividualValidationAreas(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc": zObject(map[string]*zSchema{"by": zNumber()}),
			},
			Context: zObject(map[string]*zSchema{"count": zNumber()}),
		},
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + storeValidate1Num(ev.(xs.E)["by"])}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx](xstore.ValidateSchemasOptions{
		SkipContext: true,
		SkipEvents:  true,
	}))

	s.Trigger("inc", xs.E{"by": 1})
	assert.Equal(t, storeValidate1IntCtx{Count: 1}, s.GetSnapshot().Context)
}

// JS: warns and no-ops in dev when there are no schemas
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L311
func TestStoreValidate_WarnsAndNoOpsInDevWhenThereAreNoSchemas(t *testing.T) {
	warnSpy := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[storeValidate1IntCtx]{
		Context: storeValidate1IntCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeValidate1IntCtx]{
			"inc": func(c storeValidate1IntCtx, ev xs.Event, enq *xstore.EnqueueObject[storeValidate1IntCtx]) (storeValidate1IntCtx, bool) {
				return storeValidate1IntCtx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.ValidateSchemas[storeValidate1IntCtx](xstore.ValidateSchemasOptions{
		Warn: func(args ...any) { warnSpy.Call(args...) },
	}))

	assert.Contains(t, warnSpy.Calls(), []any{
		"The \"validateSchemas\" store extension was used, but the store has no schemas to validate.",
	})

	s.Trigger("inc")
	assert.Equal(t, storeValidate1IntCtx{Count: 1}, s.GetSnapshot().Context)
}

// JS: throws a validation error for async schemas
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/validate.test.ts#L330
func TestStoreValidate_ThrowsAValidationErrorForAsyncSchemas(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[struct{}]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{"ping": zAsync(zObject(map[string]*zSchema{}))},
		},
		Context: struct{}{},
		On: map[string]xstore.StoreAssigner[struct{}]{
			"ping": func(c struct{}, ev xs.Event, enq *xstore.EnqueueObject[struct{}]) (struct{}, bool) {
				return c, true
			},
		},
	}).With(xstore.ValidateSchemas[struct{}]())

	ve := storeValidate1Thrown(t, func() { s.Trigger("ping") })
	assert.Equal(t, xstore.ReasonAsyncValidationUnsupported, ve.Reason)
	assert.Equal(t, "ping", ve.EventType)
}
