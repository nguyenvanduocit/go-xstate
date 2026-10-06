// Package tracetest replays a golden trace produced by the JS reference
// implementation (scripts/trace) against a Go actor and compares the
// snapshot after every step.
//
// Golden file: { "name", "clock": bool, "input": any, "steps": [ { "step": "start" | {"send": E} | {"advance": ms} | {"wait": ms}, "snapshot": {...} } ] }
package tracetest

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type golden struct {
	Name  string `json:"name"`
	Clock bool   `json:"clock"`
	Input any    `json:"input"`
	Steps []struct {
		Step     json.RawMessage `json:"step"`
		Snapshot map[string]any  `json:"snapshot"`
	} `json:"steps"`
}

// Run creates the actor with create (clock is nil unless the golden file was
// recorded with a SimulatedClock), starts it, replays every step and compares
// View(snapshot) with the recorded snapshot.
func Run[S xs.Snapshot](t *testing.T, goldenPath string, create func(clock xs.Clock, input any) *xs.Actor[S]) {
	t.Helper()
	raw, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	var g golden
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps, "golden file has no steps")

	var clock *xs.SimulatedClock
	var xsClock xs.Clock
	if g.Clock {
		clock = xs.NewSimulatedClock()
		xsClock = clock
	}
	actor := create(xsClock, g.Input)
	defer actor.Stop()
	actor.Start()

	for i, s := range g.Steps {
		var name string
		if json.Unmarshal(s.Step, &name) == nil {
			require.Equal(t, "start", name)
		} else {
			var step struct {
				Send    map[string]any `json:"send"`
				Advance *float64       `json:"advance"`
				Wait    *float64       `json:"wait"`
			}
			require.NoError(t, json.Unmarshal(s.Step, &step))
			switch {
			case step.Send != nil:
				actor.Send(xs.E(step.Send))
			case step.Advance != nil:
				require.NotNil(t, clock, "golden has an advance step but no clock")
				clock.Increment(time.Duration(*step.Advance * float64(time.Millisecond)))
			case step.Wait != nil:
				time.Sleep(time.Duration(*step.Wait * float64(time.Millisecond)))
			default:
				t.Fatalf("step %d: unknown step %s", i, s.Step)
			}
		}
		got := normalize(t, View(actor.GetSnapshot()))
		want := normalize(t, s.Snapshot)
		require.Equal(t, want, got, "step %d (%s)", i, s.Step)
	}
}

// View mirrors view() in scripts/trace/trace.ts.
func View(snap xs.Snapshot) map[string]any {
	out := map[string]any{
		"status": string(snap.GetStatus()),
		"output": snap.GetOutput(),
	}
	if e := snap.GetError(); e != nil {
		if err, ok := e.(error); ok {
			out["error"] = err.Error()
		} else {
			out["error"] = e
		}
	}
	v := reflect.Indirect(reflect.ValueOf(snap))
	field := func(name string) (reflect.Value, bool) {
		f := v.FieldByName(name)
		return f, f.IsValid()
	}
	if f, ok := field("Value"); ok {
		out["value"] = f.Interface()
	}
	if f, ok := field("Context"); ok {
		out["context"] = f.Interface()
	} else {
		out["context"] = nil
	}
	tags := []string{}
	if f, ok := field("Tags"); ok {
		tags = append(tags, f.Interface().([]string)...)
		sort.Strings(tags)
	}
	out["tags"] = tags
	children := []string{}
	if f, ok := field("Children"); ok {
		for _, k := range f.MapKeys() {
			children = append(children, k.String())
		}
		sort.Strings(children)
	}
	out["children"] = children
	return out
}

// normalize round-trips through JSON, as JS JSON.stringify does for the
// golden file (drops nil-able omitted fields, turns structs into objects).
func normalize(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}
