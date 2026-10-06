package xstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Keep numeric payloads inside JSON strings while snapshots pass through
// map[string]any. Otherwise an intermediate float64 can irreversibly round
// int64/uint64 values before they reach the typed decoder.
func marshalTaskValue(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		JSON string `json:"json"`
	}{JSON: string(payload)})
}

func unmarshalTaskValue[T any](encoded []byte, value *T) error {
	var envelope struct {
		JSON string `json:"json"`
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("task envelope: %w", err)
	}
	if err := jsonEOF(decoder); err != nil {
		return fmt.Errorf("task envelope: %w", err)
	}
	if envelope.JSON == "" {
		return errors.New("task envelope is missing its json payload")
	}
	decoder = json.NewDecoder(bytes.NewBufferString(envelope.JSON))
	decoder.UseNumber()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("task payload: %w", err)
	}
	if err := jsonEOF(decoder); err != nil {
		return fmt.Errorf("task payload: %w", err)
	}
	return nil
}

func jsonEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("unexpected trailing JSON value")
	}
	return nil
}

func (v taskInput[I]) MarshalJSON() ([]byte, error) { return marshalTaskValue(v.Value) }
func (v *taskInput[I]) UnmarshalJSON(encoded []byte) error {
	return unmarshalTaskValue(encoded, &v.Value)
}
func (v taskOutput[O]) MarshalJSON() ([]byte, error) { return marshalTaskValue(v.Value) }
func (v *taskOutput[O]) UnmarshalJSON(encoded []byte) error {
	return unmarshalTaskValue(encoded, &v.Value)
}

// Reuse promise execution while delivering restoration failures at startup.
// Parents start only active children, so a pending rejection must stay active
// until startup can send the error event through the usual promise transition.
type taskPromise[O any] struct{ *PromiseLogic[taskOutput[O]] }

func (l *taskPromise[O]) restoreAny(persisted any, _ *ActorScope) Snapshot {
	if snapshot, ok := persisted.(*PromiseSnapshot[taskOutput[O]]); ok && snapshot != nil {
		return snapshot
	}
	failure := func(err error) Snapshot {
		return &PromiseSnapshot[taskOutput[O]]{Status: StatusActive, Error: fmt.Errorf("restore task snapshot: %w", err)}
	}
	encoded, err := json.Marshal(persisted)
	if err != nil {
		return failure(err)
	}
	var snapshot PromiseSnapshot[taskOutput[O]]
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return failure(err)
	}
	switch snapshot.Status {
	case StatusActive, StatusStopped, StatusError:
	case StatusDone:
		var object map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &object); err != nil {
			return failure(err)
		}
		if output, ok := object["output"]; !ok || bytes.Equal(bytes.TrimSpace(output), []byte("null")) {
			return failure(errors.New("completed task is missing output"))
		}
	default:
		return failure(fmt.Errorf("invalid task status %q", snapshot.Status))
	}
	return &snapshot
}

func (l *taskPromise[O]) startAny(s Snapshot, scope *ActorScope) {
	state := s.(*PromiseSnapshot[taskOutput[O]])
	if state.Status == StatusActive && state.Error != nil {
		scope.System.relay(scope.Self, scope.Self, promiseRejectEvent{Data: state.Error})
		return
	}
	l.PromiseLogic.startAny(s, scope)
}

func (l *taskPromise[O]) newActorRef(options actorOptions) ActorRef {
	return newActor[*PromiseSnapshot[taskOutput[O]]](l, l, options)
}
