package xstate_test

import (
	"context"
	"sync"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inspect1Recorder collects inspection events; promise actors deliver events
// from other goroutines, so access is guarded.
type inspect1Recorder struct {
	mu     sync.Mutex
	events []xs.InspectionEvent
}

func (r *inspect1Recorder) Record(ev xs.InspectionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *inspect1Recorder) Events() []xs.InspectionEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]xs.InspectionEvent, len(r.events))
	copy(out, r.events)
	return out
}

// WaitForPromiseDone blocks until the recorder holds the promise actor's final
// snapshot. JS `await waitFor(...)` resumes on a microtask, after the whole
// synchronous emit chain finished; Go's WaitFor wakes on another goroutine, so
// the trailing events of that chain need an explicit wait.
func (r *inspect1Recorder) WaitForPromiseDone(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, ev := range r.Events() {
			if ev.Type != xs.InspectSnapshot {
				continue
			}
			if snap, ok := ev.Snapshot.(*xs.PromiseSnapshot[int]); ok && snap.Status == xs.StatusDone {
				return true
			}
		}
		return false
	}, 5*time.Second, time.Millisecond, "promise actor never emitted its done snapshot")
}

// inspect1Labeler maps actor session ids to the labels the JS inline
// snapshots show ("x:0", "x:1", ...). JS session ids come from a module-level
// counter, so their absolute values depend on test order; Go assigns the
// expected labels to distinct session ids in order of first appearance.
type inspect1Labeler struct {
	labels []string
	seen   map[string]string
}

func newInspect1Labeler(labels ...string) *inspect1Labeler {
	return &inspect1Labeler{labels: labels, seen: map[string]string{}}
}

func (l *inspect1Labeler) Of(ref xs.ActorRef) string {
	sid := ref.SessionID()
	if label, ok := l.seen[sid]; ok {
		return label
	}
	label := "unexpected session " + sid
	if n := len(l.seen); n < len(l.labels) {
		label = l.labels[n]
	}
	l.seen[sid] = label
	return label
}

// inspect1OnlyTypes mirrors `(ev) => [...types].includes(ev.type)`.
func inspect1OnlyTypes(types ...string) func(xs.InspectionEvent) bool {
	return func(ev xs.InspectionEvent) bool {
		for _, typ := range types {
			if ev.Type == typ {
				return true
			}
		}
		return false
	}
}

// inspect1HasType mirrors
// `expect(events).toContainEqual(expect.objectContaining({ type }))`.
func inspect1HasType(events []xs.InspectionEvent, typ string) bool {
	for _, ev := range events {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

// inspect1SimplifyEvents mirrors simplifyEvents (inspect.test.ts:16-69).
// Unhandled inspection event types map to nil, like the JS `undefined`.
func inspect1SimplifyEvents(events []xs.InspectionEvent, label *inspect1Labeler, filter func(xs.InspectionEvent) bool) []map[string]any {
	out := []map[string]any{}
	for _, ev := range events {
		if filter != nil && !filter(ev) {
			continue
		}
		switch ev.Type {
		case xs.InspectEvent:
			var sourceID any
			if ev.SourceRef != nil {
				sourceID = label.Of(ev.SourceRef)
			}
			out = append(out, map[string]any{
				"type":     ev.Type,
				"sourceId": sourceID,
				"targetId": label.Of(ev.ActorRef),
				"event":    ev.Event,
			})
		case xs.InspectActor:
			out = append(out, map[string]any{
				"type":    ev.Type,
				"actorId": label.Of(ev.ActorRef),
			})
		case xs.InspectSnapshot:
			var snapshot any = ev.Snapshot
			if xs.IsMachineSnapshot(ev.Snapshot) {
				snapshot = map[string]any{"value": ev.Snapshot.(*xs.MachineSnapshot[any]).Value}
			}
			out = append(out, map[string]any{
				"type":     ev.Type,
				"actorId":  label.Of(ev.ActorRef),
				"snapshot": snapshot,
				"event":    ev.Event,
				"status":   ev.Snapshot.GetStatus(),
			})
		case xs.InspectMicrostep:
			transitions := []map[string]any{}
			for _, tr := range ev.Transitions {
				targets := []string{}
				for _, target := range tr.Target {
					targets = append(targets, target.ID)
				}
				transitions = append(transitions, map[string]any{
					"eventType": tr.EventType,
					"target":    targets,
				})
			}
			out = append(out, map[string]any{
				"type":        ev.Type,
				"value":       ev.Snapshot.(*xs.MachineSnapshot[any]).Value,
				"event":       ev.Event,
				"transitions": transitions,
			})
		case xs.InspectAction:
			out = append(out, map[string]any{
				"type":   ev.Type,
				"action": *ev.Action,
			})
		default:
			out = append(out, nil)
		}
	}
	return out
}

// JS: inspect > the .inspect option can observe inspection events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L72
func TestInspect_TheInspectOptionCanObserveInspectionEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
			{Key: "c"},
		},
	})

	rec := &inspect1Recorder{}

	actor := xs.CreateActor(machine, xs.WithInspect(rec.Record))
	actor.Start()

	actor.Send(xs.Ev("NEXT"))
	actor.Send(xs.Ev("NEXT"))

	label := newInspect1Labeler("x:0")
	assert.Equal(t, []map[string]any{
		{"actorId": "x:0", "type": xs.InspectActor},
		{"event": xs.InitEvent{Input: nil}, "sourceId": nil, "targetId": "x:0", "type": xs.InspectEvent},
		{"actorId": "x:0", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "a"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.Ev("NEXT"), "sourceId": nil, "targetId": "x:0", "type": xs.InspectEvent},
		{"actorId": "x:0", "event": xs.Ev("NEXT"), "snapshot": map[string]any{"value": "b"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.Ev("NEXT"), "sourceId": nil, "targetId": "x:0", "type": xs.InspectEvent},
		{"actorId": "x:0", "event": xs.Ev("NEXT"), "snapshot": map[string]any{"value": "c"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
	}, inspect1SimplifyEvents(rec.Events(), label, inspect1OnlyTypes(xs.InspectActor, xs.InspectEvent, xs.InspectSnapshot)))
	assert.Equal(t, "x:0", label.Of(actor))
}

// JS: inspect > can inspect communications between actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L173
func TestInspect_CanInspectCommunicationsBetweenActors(t *testing.T) {
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"loadChild": {{Target: "loading"}}}},
			{
				Key: "loading",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
						return 42, nil
					}),
					OnDone: xs.Transitions{{
						Target:  "loaded",
						Actions: xs.Actions{xs.SendParent(xs.Ev("toParent"))},
					}},
				}},
			},
			{Key: "loaded", Type: xs.Final},
		},
	})

	rec := &inspect1Recorder{}

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting"},
			{Key: "success"},
		},
		Invoke: []xs.InvokeConfig{{
			Logic: childMachine,
			ID:    "child",
			OnDone: xs.Transitions{{
				Target: ".success",
				Actions: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
					_ = rec
				})},
			}},
		}},
		On: map[string]xs.Transitions{
			"load": {{Actions: xs.Actions{xs.SendTo("child", xs.Ev("loadChild"))}}},
		},
	})

	actor := xs.CreateActor(parentMachine, xs.WithInspectObserver(xs.Observer[xs.InspectionEvent]{
		Next: rec.Record,
	}))

	actor.Start()
	actor.Send(xs.Ev("load"))

	_, err := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool {
		return s.Value == "success"
	}).Wait()
	require.NoError(t, err)
	rec.WaitForPromiseDone(t)

	label := newInspect1Labeler("x:1", "x:2", "x:3")
	assert.Equal(t, []map[string]any{
		{"actorId": "x:1", "type": xs.InspectActor},
		{"actorId": "x:2", "type": xs.InspectActor},
		{"event": xs.InitEvent{Input: nil}, "sourceId": nil, "targetId": "x:1", "type": xs.InspectEvent},
		{"event": xs.InitEvent{Input: nil}, "sourceId": "x:1", "targetId": "x:2", "type": xs.InspectEvent},
		{"actorId": "x:2", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "start"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"actorId": "x:1", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "waiting"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.Ev("load"), "sourceId": nil, "targetId": "x:1", "type": xs.InspectEvent},
		{"event": xs.Ev("loadChild"), "sourceId": "x:1", "targetId": "x:2", "type": xs.InspectEvent},
		{"actorId": "x:3", "type": xs.InspectActor},
		{"event": xs.InitEvent{Input: nil}, "sourceId": "x:2", "targetId": "x:3", "type": xs.InspectEvent},
		{
			"actorId":  "x:3",
			"event":    xs.InitEvent{Input: nil},
			"snapshot": &xs.PromiseSnapshot[int]{Error: nil, Input: nil, Output: 0, Status: xs.StatusActive},
			"status":   xs.StatusActive,
			"type":     xs.InspectSnapshot,
		},
		{"actorId": "x:2", "event": xs.Ev("loadChild"), "snapshot": map[string]any{"value": "loading"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"actorId": "x:1", "event": xs.Ev("load"), "snapshot": map[string]any{"value": "waiting"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.PromiseResolveEvent{Data: 42}, "sourceId": "x:3", "targetId": "x:3", "type": xs.InspectEvent},
		{"event": xs.DoneActorEvent{ActorID: "0.(machine).loading", Output: 42}, "sourceId": "x:3", "targetId": "x:2", "type": xs.InspectEvent},
		{"event": xs.Ev("toParent"), "sourceId": "x:2", "targetId": "x:1", "type": xs.InspectEvent},
		{"actorId": "x:1", "event": xs.Ev("toParent"), "snapshot": map[string]any{"value": "waiting"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.DoneActorEvent{ActorID: "child", Output: nil}, "sourceId": "x:2", "targetId": "x:1", "type": xs.InspectEvent},
		{"actorId": "x:1", "event": xs.DoneActorEvent{ActorID: "child", Output: nil}, "snapshot": map[string]any{"value": "success"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"actorId": "x:2", "event": xs.DoneActorEvent{ActorID: "0.(machine).loading", Output: 42}, "snapshot": map[string]any{"value": "loaded"}, "status": xs.StatusDone, "type": xs.InspectSnapshot},
		{
			"actorId":  "x:3",
			"event":    xs.PromiseResolveEvent{Data: 42},
			"snapshot": &xs.PromiseSnapshot[int]{Error: nil, Input: nil, Output: 42, Status: xs.StatusDone},
			"status":   xs.StatusDone,
			"type":     xs.InspectSnapshot,
		},
	}, inspect1SimplifyEvents(rec.Events(), label, inspect1OnlyTypes(xs.InspectActor, xs.InspectEvent, xs.InspectSnapshot)))
	assert.Equal(t, "x:1", label.Of(actor))
}

// JS: inspect > can inspect microsteps from always events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L450
func TestInspect_CanInspectMicrostepsFromAlwaysEvents(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Initial: "counting",
		States: xs.States{
			{
				Key: "counting",
				Always: xs.Transitions{
					{
						Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count == 3 }),
						Target: "done",
					},
					{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: a.Context.Count + 1}
					})}},
				},
			},
			{Key: "done"},
		},
	})

	rec := &inspect1Recorder{}

	actor := xs.CreateActor(machine, xs.WithInspect(rec.Record)).Start()

	events := rec.Events()
	rootID := actor.SessionID()
	require.Len(t, events, 7)

	// Common fields of every event: actorRef { id: "x:4" }, rootId "x:4".
	for i, ev := range events {
		require.NotNil(t, ev.ActorRef, "event %d", i)
		assert.Equal(t, rootID, ev.ActorRef.SessionID(), "event %d actorRef", i)
		assert.Equal(t, rootID, ev.RootID, "event %d rootId", i)
	}

	// snapshot: { children: {}, context, error: undefined, historyValue: {},
	// output: undefined, status: "active", tags: [], value }
	assertSnap := func(i int, count int, value string) {
		t.Helper()
		snap, ok := events[i].Snapshot.(*xs.MachineSnapshot[ctx])
		require.True(t, ok, "event %d snapshot type", i)
		assert.Empty(t, snap.Children, "event %d children", i)
		assert.Equal(t, ctx{Count: count}, snap.Context, "event %d context", i)
		assert.Nil(t, snap.Error, "event %d error", i)
		assert.Empty(t, snap.HistoryValue, "event %d historyValue", i)
		assert.Nil(t, snap.Output, "event %d output", i)
		assert.Equal(t, xs.StatusActive, snap.Status, "event %d status", i)
		assert.Empty(t, snap.Tags, "event %d tags", i)
		assert.Equal(t, value, snap.Value, "event %d value", i)
	}

	// 0: @xstate.actor
	assert.Equal(t, xs.InspectActor, events[0].Type)

	// 1-3: @xstate.microstep for the targetless assign transition
	for i := 1; i <= 3; i++ {
		count := i
		ev := events[i]
		assert.Equal(t, xs.InspectMicrostep, ev.Type, "event %d type", i)
		assert.Equal(t, xs.InitEvent{Input: nil}, ev.Event, "event %d event", i)
		require.Len(t, ev.Transitions, 1, "event %d transitions", i)
		tr := ev.Transitions[0]
		assert.Len(t, tr.Actions, 1, "event %d actions", i)
		assert.Equal(t, "", tr.EventType, "event %d eventType", i)
		assert.Nil(t, tr.Guard, "event %d guard", i)
		assert.False(t, tr.Reenter, "event %d reenter", i)
		require.NotNil(t, tr.Source, "event %d source", i)
		assert.Equal(t, "(machine).counting", tr.Source.ID, "event %d source", i)
		assert.Nil(t, tr.Target, "event %d target", i)
		assertSnap(i, count, "counting")
	}

	// 4: @xstate.microstep for the guarded transition to "done"
	{
		ev := events[4]
		assert.Equal(t, xs.InspectMicrostep, ev.Type)
		assert.Equal(t, xs.InitEvent{Input: nil}, ev.Event)
		require.Len(t, ev.Transitions, 1)
		tr := ev.Transitions[0]
		assert.Empty(t, tr.Actions)
		assert.Equal(t, "", tr.EventType)
		assert.NotNil(t, tr.Guard)
		assert.False(t, tr.Reenter)
		require.NotNil(t, tr.Source)
		assert.Equal(t, "(machine).counting", tr.Source.ID)
		require.Len(t, tr.Target, 1)
		assert.Equal(t, "(machine).done", tr.Target[0].ID)
		assertSnap(4, 3, "done")
	}

	// 5: @xstate.event
	assert.Equal(t, xs.InspectEvent, events[5].Type)
	assert.Equal(t, xs.InitEvent{Input: nil}, events[5].Event)
	assert.Nil(t, events[5].SourceRef)

	// 6: @xstate.snapshot
	assert.Equal(t, xs.InspectSnapshot, events[6].Type)
	assert.Equal(t, xs.InitEvent{Input: nil}, events[6].Event)
	assertSnap(6, 3, "done")
}

// JS: inspect > can inspect microsteps from raised events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L672
func TestInspect_CanInspectMicrostepsFromRaisedEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("to_b"))},
				On:    map[string]xs.Transitions{"to_b": {{Target: "b"}}},
			},
			{
				Key:   "b",
				Entry: xs.Actions{xs.Raise(xs.Ev("to_c"))},
				On:    map[string]xs.Transitions{"to_c": {{Target: "c"}}},
			},
			{Key: "c"},
		},
	})

	rec := &inspect1Recorder{}

	actor := xs.CreateActor(machine, xs.WithInspect(rec.Record)).Start()

	label := newInspect1Labeler("x:5")
	assert.Equal(t, []map[string]any{
		{"actorId": "x:5", "type": xs.InspectActor},
		{
			"event":       xs.Ev("to_b"),
			"transitions": []map[string]any{{"eventType": "to_b", "target": []string{"(machine).b"}}},
			"type":        xs.InspectMicrostep,
			"value":       "b",
		},
		{
			"event":       xs.Ev("to_c"),
			"transitions": []map[string]any{{"eventType": "to_c", "target": []string{"(machine).c"}}},
			"type":        xs.InspectMicrostep,
			"value":       "c",
		},
		{"event": xs.InitEvent{Input: nil}, "sourceId": nil, "targetId": "x:5", "type": xs.InspectEvent},
		{
			"action": xs.InspectedAction{
				Params: map[string]any{"delay": nil, "event": xs.Ev("to_b"), "id": nil},
				Type:   "xstate.raise",
			},
			"type": xs.InspectAction,
		},
		{
			"action": xs.InspectedAction{
				Params: map[string]any{"delay": nil, "event": xs.Ev("to_c"), "id": nil},
				Type:   "xstate.raise",
			},
			"type": xs.InspectAction,
		},
		{"actorId": "x:5", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "c"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
	}, inspect1SimplifyEvents(rec.Events(), label, nil))
	assert.Equal(t, "x:5", label.Of(actor))
}

// JS: inspect > should inspect microsteps for normal transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L783
func TestInspect_ShouldInspectMicrostepsForNormalTransitions(t *testing.T) {
	rec := &inspect1Recorder{}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EV": {{Target: "b"}}}},
			{Key: "b"},
		},
	})
	actorRef := xs.CreateActor(machine, xs.WithInspect(rec.Record)).Start()
	actorRef.Send(xs.Ev("EV"))

	label := newInspect1Labeler("x:6")
	assert.Equal(t, []map[string]any{
		{"actorId": "x:6", "type": xs.InspectActor},
		{"event": xs.InitEvent{Input: nil}, "sourceId": nil, "targetId": "x:6", "type": xs.InspectEvent},
		{"actorId": "x:6", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "a"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.Ev("EV"), "sourceId": nil, "targetId": "x:6", "type": xs.InspectEvent},
		{
			"event":       xs.Ev("EV"),
			"transitions": []map[string]any{{"eventType": "EV", "target": []string{"(machine).b"}}},
			"type":        xs.InspectMicrostep,
			"value":       "b",
		},
		{"actorId": "x:6", "event": xs.Ev("EV"), "snapshot": map[string]any{"value": "b"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
	}, inspect1SimplifyEvents(rec.Events(), label, nil))
	assert.Equal(t, "x:6", label.Of(actorRef))
}

// JS: inspect > should inspect microsteps for eventless/always transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L862
func TestInspect_ShouldInspectMicrostepsForEventlessAlwaysTransitions(t *testing.T) {
	rec := &inspect1Recorder{}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EV": {{Target: "b"}}}},
			{Key: "b", Always: xs.Transitions{{Target: "c"}}},
			{Key: "c"},
		},
	})
	actorRef := xs.CreateActor(machine, xs.WithInspect(rec.Record)).Start()
	actorRef.Send(xs.Ev("EV"))

	label := newInspect1Labeler("x:7")
	assert.Equal(t, []map[string]any{
		{"actorId": "x:7", "type": xs.InspectActor},
		{"event": xs.InitEvent{Input: nil}, "sourceId": nil, "targetId": "x:7", "type": xs.InspectEvent},
		{"actorId": "x:7", "event": xs.InitEvent{Input: nil}, "snapshot": map[string]any{"value": "a"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
		{"event": xs.Ev("EV"), "sourceId": nil, "targetId": "x:7", "type": xs.InspectEvent},
		{
			"event":       xs.Ev("EV"),
			"transitions": []map[string]any{{"eventType": "EV", "target": []string{"(machine).b"}}},
			"type":        xs.InspectMicrostep,
			"value":       "b",
		},
		{
			"event":       xs.Ev("EV"),
			"transitions": []map[string]any{{"eventType": "", "target": []string{"(machine).c"}}},
			"type":        xs.InspectMicrostep,
			"value":       "c",
		},
		{"actorId": "x:7", "event": xs.Ev("EV"), "snapshot": map[string]any{"value": "c"}, "status": xs.StatusActive, "type": xs.InspectSnapshot},
	}, inspect1SimplifyEvents(rec.Events(), label, nil))
	assert.Equal(t, "x:7", label.Of(actorRef))
}

// JS: inspect > should inspect actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L957
func TestInspect_ShouldInspectActions(t *testing.T) {
	rec := &inspect1Recorder{}

	machine := xs.NewSetup[any](xs.Implementations{
		Actions: map[string]xs.Action{
			"enter1":       xs.ActionFunc(func(_ xs.ActionArgs[any]) {}),
			"exit1":        xs.ActionFunc(func(_ xs.ActionArgs[any]) {}),
			"stringAction": xs.ActionFunc(func(_ xs.ActionArgs[any]) {}),
			"namedAction":  xs.ActionFunc(func(_ xs.ActionArgs[any]) {}),
		},
	}).CreateMachine(xs.MachineConfig[any]{
		Entry:   xs.Actions{xs.ActionRef{Type: "enter1"}},
		Exit:    xs.Actions{xs.ActionRef{Type: "exit1"}},
		Initial: "loading",
		States: xs.States{
			{
				Key: "loading",
				On: map[string]xs.Transitions{
					"event": {{
						Target: "done",
						Actions: xs.Actions{
							xs.ActionRef{Type: "stringAction"},
							xs.ActionRef{Type: "namedAction", Params: map[string]any{"foo": "bar"}},
							xs.ActionFunc(func(_ xs.ActionArgs[any]) {
								/* inline */
							}),
						},
					}},
				},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine, xs.WithInspect(func(ev xs.InspectionEvent) {
		if ev.Type == xs.InspectAction {
			rec.Record(ev)
		}
	}))

	actor.Start()
	actor.Send(xs.Ev("event"))

	label := newInspect1Labeler()
	assert.Equal(t, []map[string]any{
		{"action": xs.InspectedAction{Params: nil, Type: "enter1"}, "type": xs.InspectAction},
		{"action": xs.InspectedAction{Params: nil, Type: "stringAction"}, "type": xs.InspectAction},
		{"action": xs.InspectedAction{Params: map[string]any{"foo": "bar"}, Type: "namedAction"}, "type": xs.InspectAction},
		{"action": xs.InspectedAction{Params: nil, Type: "(anonymous)"}, "type": xs.InspectAction},
		{"action": xs.InspectedAction{Params: nil, Type: "exit1"}, "type": xs.InspectAction},
	}, inspect1SimplifyEvents(rec.Events(), label, inspect1OnlyTypes(xs.InspectAction)))
}

// JS: inspect > @xstate.microstep inspection events should report no transitions if an unknown event was sent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L1047
func TestInspect_MicrostepInspectionEventsShouldReportNoTransitionsIfAnUnknownEventWasSent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	// expect.assertions(1): exactly one microstep event, with zero transitions.
	var mu sync.Mutex
	microsteps := 0

	actor := xs.CreateActor(machine, xs.WithInspect(func(ev xs.InspectionEvent) {
		if ev.Type == xs.InspectMicrostep {
			mu.Lock()
			microsteps++
			mu.Unlock()
			assert.Equal(t, 0, len(ev.Transitions))
		}
	}))

	actor.Start()
	actor.Send(xs.Ev("any"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, microsteps, "expect.assertions(1)")
}

// JS: inspect > actor.system.inspect(…) can inspect actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L1063
func TestInspect_ActorSystemInspectCanInspectActors(t *testing.T) {
	actor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))
	var events []xs.InspectionEvent

	actor.System().Inspect(func(ev xs.InspectionEvent) {
		events = append(events, ev)
	})

	actor.Start()

	assert.True(t, inspect1HasType(events, xs.InspectEvent), "events should contain a %s event", xs.InspectEvent)
	assert.True(t, inspect1HasType(events, xs.InspectSnapshot), "events should contain a %s event", xs.InspectSnapshot)
}

// JS: inspect > actor.system.inspect(…) can inspect actors (observer)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L1085
func TestInspect_ActorSystemInspectCanInspectActorsObserver(t *testing.T) {
	actor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))
	var events []xs.InspectionEvent

	actor.System().InspectObserver(xs.Observer[xs.InspectionEvent]{
		Next: func(ev xs.InspectionEvent) {
			events = append(events, ev)
		},
	})

	actor.Start()

	assert.True(t, inspect1HasType(events, xs.InspectEvent), "events should contain a %s event", xs.InspectEvent)
	assert.True(t, inspect1HasType(events, xs.InspectSnapshot), "events should contain a %s event", xs.InspectSnapshot)
}

// JS: inspect > actor.system.inspect(…) can be unsubscribed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L1109
func TestInspect_ActorSystemInspectCanBeUnsubscribed(t *testing.T) {
	actor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))
	var events []xs.InspectionEvent

	sub := actor.System().Inspect(func(ev xs.InspectionEvent) {
		events = append(events, ev)
	})

	actor.Start()

	assert.Equal(t, 2, len(events))

	events = events[:0]

	sub.Unsubscribe()

	actor.Send(xs.Ev("someEvent"))

	assert.Equal(t, 0, len(events))
}

// JS: inspect > actor.system.inspect(…) can be unsubscribed (observer)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/inspect.test.ts#L1130
func TestInspect_ActorSystemInspectCanBeUnsubscribedObserver(t *testing.T) {
	actor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))
	var events []xs.InspectionEvent

	sub := actor.System().InspectObserver(xs.Observer[xs.InspectionEvent]{
		Next: func(ev xs.InspectionEvent) {
			events = append(events, ev)
		},
	})

	actor.Start()

	assert.Equal(t, 2, len(events))

	events = events[:0]

	sub.Unsubscribe()

	actor.Send(xs.Ev("someEvent"))

	assert.Equal(t, 0, len(events))
}
