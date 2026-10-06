package xstate

import (
	"encoding/json"
	"testing"
)

func checkTaskNumbers[T comparable](t *testing.T, want T) {
	t.Helper()
	encoded, err := json.Marshal(taskInput[T]{Value: want})
	if err != nil {
		t.Fatal(err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatal(err)
	}
	got, err := decodeTaskInput[T](generic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != want {
		t.Fatalf("input corrupted: got %v want %v", got.Value, want)
	}
	encoded, err = json.Marshal(&PromiseSnapshot[taskOutput[T]]{Status: StatusDone, Output: taskOutput[T]{Value: want}})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatal(err)
	}
	snapshot := convertSnapshot[*PromiseSnapshot[taskOutput[T]]](generic)
	if snapshot == nil || snapshot.Output.Value != want {
		t.Fatalf("output corrupted: got %+v want %v", snapshot, want)
	}
}

func TestTaskJSONPreservesLargeIntegers(t *testing.T) {
	t.Run("above float precision", func(t *testing.T) { checkTaskNumbers(t, int64(9007199254740993)) })
	t.Run("max int64", func(t *testing.T) { checkTaskNumbers(t, int64(9223372036854775807)) })
	t.Run("max uint64", func(t *testing.T) { checkTaskNumbers(t, uint64(18446744073709551615)) })
}

func TestTaskEnvelopeRejectsMalformedPayloads(t *testing.T) {
	for _, encoded := range []string{`{}`, `null`, `{"json":""}`, `{"json":null}`, `{"json":3}`, `{"json":"1 2"}`, `{"json":"1 garbage"}`, `{"json":"1","extra":true}`, `{"json":"\"not an int\""}`} {
		t.Run(encoded, func(t *testing.T) {
			var output taskOutput[int]
			if err := json.Unmarshal([]byte(encoded), &output); err == nil {
				t.Fatal("malformed envelope accepted")
			}
		})
	}
}

func TestTaskRestoreReportsMalformedOutput(t *testing.T) {
	logic := &taskPromise[int]{}
	for _, output := range []any{map[string]any{"json": "1 garbage"}, map[string]any{"json": "\"not an int\""}, nil} {
		snapshot := logic.restoreAny(map[string]any{"status": "done", "output": output}, nil)
		if snapshot.GetStatus() != StatusActive || snapshot.GetError() == nil {
			t.Fatalf("malformed output not reported: %+v", snapshot)
		}
	}
	snapshot := logic.restoreAny(map[string]any{"status": "done"}, nil)
	if snapshot.GetStatus() != StatusActive || snapshot.GetError() == nil {
		t.Fatal("missing output accepted")
	}
}

func TestTaskEnvelopePreservesNullAndZero(t *testing.T) {
	checkTaskNumbers(t, int64(0))
	checkTaskNumbers(t, (*int)(nil))
	encoded, err := json.Marshal(taskInput[any]{Value: nil})
	if err != nil {
		t.Fatal(err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatal(err)
	}
	got, err := decodeTaskInput[any](generic)
	if err != nil || got.Value != nil {
		t.Fatalf("nil lost: %+v %v", got, err)
	}
}

func TestTaskEnvelopePreservesInterfaceNumberDigits(t *testing.T) {
	encoded, err := json.Marshal(taskInput[any]{Value: int64(9007199254740993)})
	if err != nil {
		t.Fatal(err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatal(err)
	}
	got, err := decodeTaskInput[any](generic)
	if err != nil {
		t.Fatal(err)
	}
	number, ok := got.Value.(json.Number)
	if !ok || number.String() != "9007199254740993" {
		t.Fatalf("number lost: %#v", got.Value)
	}
}

func TestTaskEnvelopeReportsMarshalFailure(t *testing.T) {
	if _, err := json.Marshal(taskInput[chan int]{Value: make(chan int)}); err == nil {
		t.Fatal("channel serialized")
	}
}
