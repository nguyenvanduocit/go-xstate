package store

import (
	"fmt"
	"log"
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// ValidationReason mirrors StoreValidationError['reason'].
type ValidationReason string

const (
	ReasonInvalidContext             ValidationReason = "invalidContext"
	ReasonInvalidEvent               ValidationReason = "invalidEvent"
	ReasonInvalidEmitted             ValidationReason = "invalidEmitted"
	ReasonUnknownEvent               ValidationReason = "unknownEvent"
	ReasonUnknownEmitted             ValidationReason = "unknownEmitted"
	ReasonAsyncValidationUnsupported ValidationReason = "asyncValidationUnsupported"
)

// StoreValidationError mirrors StoreValidationError. ValidateSchemas panics
// with a *StoreValidationError; Error() returns the JS message
// (e.g. `Invalid event "inc"`).
type StoreValidationError struct {
	Reason    ValidationReason
	EventType string
	Context   any
	Payload   any
	Issues    []SchemaIssue
}

func (e *StoreValidationError) Error() string {
	switch e.Reason {
	case ReasonInvalidContext:
		return "Invalid context"
	case ReasonInvalidEvent:
		return fmt.Sprintf("Invalid event %q", e.EventType)
	case ReasonInvalidEmitted:
		return fmt.Sprintf("Invalid emitted event %q", e.EventType)
	case ReasonUnknownEvent:
		return fmt.Sprintf("Unknown event %q", e.EventType)
	case ReasonUnknownEmitted:
		return fmt.Sprintf("Unknown emitted event %q", e.EventType)
	case ReasonAsyncValidationUnsupported:
		if e.EventType != "" {
			return fmt.Sprintf("Async schema validation is unsupported for event %q", e.EventType)
		}
		return "Async schema validation is unsupported"
	}
	return string(e.Reason)
}

// IsStoreValidationError mirrors isStoreValidationError(value).
func IsStoreValidationError(v any) bool {
	_, ok := v.(*StoreValidationError)
	return ok
}

// UnknownPolicy mirrors the 'throw' | 'ignore' options.
type UnknownPolicy string

const (
	UnknownThrow  UnknownPolicy = "throw" // default ("" behaves as "throw")
	UnknownIgnore UnknownPolicy = "ignore"
)

// ValidateSchemasOptions mirrors ValidateSchemasOptions. The JS booleans
// default to true, so Go uses Skip* fields: `{ context: false }` →
// SkipContext: true.
type ValidateSchemasOptions struct {
	SkipContext    bool
	SkipEvents     bool
	SkipEmitted    bool
	UnknownEvents  UnknownPolicy
	UnknownEmitted UnknownPolicy
	// Warn receives the development warning JS writes with console.warn
	// (store without schemas). nil: log.Println.
	Warn func(args ...any)
}

// getPayload mirrors `const { type, ...payload } = event`.
func getPayload(event xs.Event) map[string]any {
	payload := map[string]any{}
	if e, ok := event.(xs.E); ok {
		for k, v := range e {
			if k != "type" {
				payload[k] = v
			}
		}
	}
	return payload
}

func hasAnySchemas(s *StoreSchemas) bool {
	return s != nil && (s.Context != nil || s.Events != nil || s.Emitted != nil)
}

func validateSchema(schema Schema, value any, errTemplate StoreValidationError) {
	result, promise := schema.Validate(value)
	if promise != nil {
		e := errTemplate
		e.Reason = ReasonAsyncValidationUnsupported
		panic(&e)
	}
	if len(result.Issues) > 0 {
		e := errTemplate
		e.Issues = result.Issues
		panic(&e)
	}
}

func validateEvent(event xs.Event, schemas *StoreSchemas, eventTypes []string, unknown UnknownPolicy) {
	if schemas.Events == nil {
		return
	}
	payload := getPayload(event)
	schema := schemas.Events[event.EventType()]
	if schema == nil {
		if slices.Contains(eventTypes, event.EventType()) {
			return
		}
		if unknown != UnknownIgnore {
			panic(&StoreValidationError{Reason: ReasonUnknownEvent, EventType: event.EventType(), Payload: payload})
		}
		return
	}
	validateSchema(schema, payload, StoreValidationError{Reason: ReasonInvalidEvent, EventType: event.EventType(), Payload: payload})
}

func validateContext(ctx any, schemas *StoreSchemas) {
	if schemas.Context == nil {
		return
	}
	validateSchema(schemas.Context, ctx, StoreValidationError{Reason: ReasonInvalidContext, Context: ctx})
}

func validateEmitted(effect xs.Event, schemas *StoreSchemas, unknown UnknownPolicy) {
	if schemas.Emitted == nil {
		return
	}
	payload := getPayload(effect)
	schema := schemas.Emitted[effect.EventType()]
	if schema == nil {
		if unknown != UnknownIgnore {
			panic(&StoreValidationError{Reason: ReasonUnknownEmitted, EventType: effect.EventType(), Payload: payload})
		}
		return
	}
	validateSchema(schema, payload, StoreValidationError{Reason: ReasonInvalidEmitted, EventType: effect.EventType(), Payload: payload})
}

// ValidateSchemas mirrors validateSchemas(options?) from
// @xstate/store/validate.
func ValidateSchemas[C any](opts ...ValidateSchemasOptions) StoreExtension[C] {
	var options ValidateSchemasOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	return func(logic StoreLogic[C]) StoreLogic[C] {
		schemas := logic.Schemas
		if !hasAnySchemas(schemas) {
			warn := options.Warn
			if warn == nil {
				warn = func(args ...any) { log.Println(args...) }
			}
			warn(`The "validateSchemas" store extension was used, but the store has no schemas to validate.`)
			return logic
		}

		enhanced := logic
		enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] {
			snapshot := logic.GetInitialSnapshot()
			if !options.SkipContext {
				validateContext(snapshot.Context, schemas)
			}
			return snapshot
		}
		enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
			if !options.SkipEvents {
				validateEvent(event, schemas, logic.EventTypes, options.UnknownEvents)
			}
			result := logic.Transition(snapshot, event)
			if !options.SkipContext {
				validateContext(result.Snapshot.Context, schemas)
			}
			if !options.SkipEmitted {
				for _, effect := range result.Effects {
					if effect.Run == nil && effect.Emitted != nil {
						validateEmitted(effect.Emitted, schemas, options.UnknownEmitted)
					}
				}
			}
			return result
		}
		return enhanced
	}
}
