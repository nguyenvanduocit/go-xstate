package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// JS: ContextFrom > should return context of a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L17
func TestTypeHelpers_ContextFrom_ShouldReturnContextOfAMachine(t *testing.T) {
	t.Skip("N/A: type-level only — ContextFrom<typeof machine> accepts {counter: 100}, @ts-expect-error on extra key `other` and on {completely: 'invalid'}; no runtime expectations")
}

// JS: EventFrom > should return events for a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L44
func TestTypeHelpers_EventFrom_ShouldReturnEventsForAMachine(t *testing.T) {
	t.Skip("N/A: type-level only — EventFrom<typeof machine> accepts UPDATE_NAME/UPDATE_AGE/ANOTHER_EVENT, @ts-expect-error on UNKNOWN_EVENT; no runtime expectations")
}

// JS: EventFrom > should return events for an interpreter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L67
func TestTypeHelpers_EventFrom_ShouldReturnEventsForAnInterpreter(t *testing.T) {
	t.Skip("N/A: type-level only — EventFrom<typeof service> accepts UPDATE_NAME/UPDATE_AGE/ANOTHER_EVENT, @ts-expect-error on UNKNOWN_EVENT; only runtime call is an unstarted createActor, no runtime expectations")
}

// JS: MachineImplementationsFrom > should return implementations for a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L94
func TestTypeHelpers_MachineImplementationsFrom_ShouldReturnImplementationsForAMachine(t *testing.T) {
	t.Skip("N/A: type-level only — MachineImplementationsFrom<typeof machine> accepts actions (plain fn and assign with typed context.count / event.type 'FOO'|'BAR'), @ts-expect-error on 100; no runtime expectations")
}

// JS: StateValueFrom > should return any from a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L140
func TestTypeHelpers_StateValueFrom_ShouldReturnAnyFromAMachine(t *testing.T) {
	t.Skip("N/A: type-level only — StateValueFrom<typeof machine> of an empty machine accepts any string; no runtime expectations")
}

// JS: SnapshotFrom > should return state type from a service that has concrete event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L150
func TestTypeHelpers_SnapshotFrom_ShouldReturnStateTypeFromAServiceThatHasConcreteEventType(t *testing.T) {
	// JS types.events {type: 'FOO'} has no Go counterpart (events are untyped xs.Event).
	service := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))

	// SnapshotFrom<typeof service> → the actor's typed snapshot; Go enforces this at compile time.
	acceptState := func(_ *xs.MachineSnapshot[any]) {}

	acceptState(service.GetSnapshot())
	// @ts-expect-error acceptState("isn't any") — a Go compile error, not expressible at runtime.
}

// JS: SnapshotFrom > should return state from a machine without context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L166
func TestTypeHelpers_SnapshotFrom_ShouldReturnStateFromAMachineWithoutContext(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{})

	acceptState := func(_ *xs.MachineSnapshot[any]) {}

	acceptState(xs.CreateActor(machine).GetSnapshot())
	// @ts-expect-error acceptState("isn't any") — a Go compile error, not expressible at runtime.
}

// JS: SnapshotFrom > should return state from a machine with context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L176
func TestTypeHelpers_SnapshotFrom_ShouldReturnStateFromAMachineWithContext(t *testing.T) {
	type ctx struct{ Counter int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
	})

	acceptState := func(_ *xs.MachineSnapshot[ctx]) {}

	acceptState(xs.CreateActor(machine).GetSnapshot())
	// @ts-expect-error acceptState("isn't any") — a Go compile error, not expressible at runtime.
}

// JS: ActorRefFrom > should return `ActorRef` based on actor logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L192
func TestTypeHelpers_ActorRefFrom_ShouldReturnActorRefBasedOnActorLogic(t *testing.T) {
	// ActorLogic<Snapshot<undefined>, { type: 'TEST' }>
	logic := &xs.Logic[*xs.BasicSnapshot[any]]{
		Transition: func(state *xs.BasicSnapshot[any], _ xs.Event, _ *xs.ActorScope) *xs.BasicSnapshot[any] {
			return state
		},
		GetInitialSnapshot: func(_ *xs.ActorScope, _ any) *xs.BasicSnapshot[any] {
			return &xs.BasicSnapshot[any]{
				Status: xs.StatusActive,
				Output: nil,
				Error:  nil,
			}
		},
		GetPersistedSnapshot: func(s *xs.BasicSnapshot[any]) any { return s },
	}

	// ActorRefFrom<typeof logic> → *xs.Actor[*xs.BasicSnapshot[any]]
	acceptActorRef := func(actorRef *xs.Actor[*xs.BasicSnapshot[any]]) {
		actorRef.Send(xs.Ev("TEST"))
	}

	acceptActorRef(xs.CreateActor(logic).Start())
}

// JS: tags > derives string from StateMachine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/typeHelpers.test.ts#L212
func TestTypeHelpers_Tags_DerivesStringFromStateMachine(t *testing.T) {
	t.Skip("N/A: type-level only — TagsFrom<typeof machine> of an untyped machine accepts any string ('a'..'d'); no runtime expectations")
}
