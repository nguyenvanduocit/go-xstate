package xstate

// Inspection event types.
const (
	InspectSnapshot   = "@xstate.snapshot"
	InspectTransition = "@xstate.transition"
	InspectMicrostep  = "@xstate.microstep"
	InspectAction     = "@xstate.action"
	InspectEvent      = "@xstate.event"
	InspectActor      = "@xstate.actor"
)

// InspectedAction mirrors `action` of an @xstate.action inspection event.
type InspectedAction struct {
	Type   string
	Params any
}

// InspectionEvent is one struct for all JS inspection event variants; Type
// selects which fields are set.
type InspectionEvent struct {
	Type        string
	RootID      string
	ActorRef    ActorRef
	SourceRef   ActorRef // @xstate.event only; nil for external events
	Event       Event    // snapshot, transition, microstep, event
	Snapshot    Snapshot // snapshot, transition, microstep
	Action      *InspectedAction
	Transitions []*TransitionDefinition // microstep
}
