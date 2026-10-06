package graph_test

import "testing"

// JS: getShortestPath types > `getEvents` should be allowed to return a mutable array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L5
func TestTypes_GetShortestPathTypes_GetEventsShouldBeAllowedToReturnAMutableArray(t *testing.T) {
	t.Skip("N/A: type-level only — checked that a mutable event array type-checks as `events`")
}

// JS: getShortestPath types > `getEvents` should be allowed to return a readonly array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L21
func TestTypes_GetShortestPathTypes_GetEventsShouldBeAllowedToReturnAReadonlyArray(t *testing.T) {
	t.Skip("N/A: type-level only — checked that a readonly event array type-checks as `events`")
}

// JS: getShortestPath types > `events` should allow known event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L37
func TestTypes_GetShortestPathTypes_EventsShouldAllowKnownEvent(t *testing.T) {
	t.Skip("N/A: type-level only — checked that a known event with payload type-checks in `events`")
}

// JS: getShortestPath types > `events` should not require all event types (array literal expression)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L54
func TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesArrayLiteral(t *testing.T) {
	t.Skip("N/A: type-level only — checked that `events` may list a subset of event types (array literal)")
}

// JS: getShortestPath types > `events` should not require all event types (tuple)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L66
func TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesTuple(t *testing.T) {
	t.Skip("N/A: type-level only — checked that `events` may list a subset of event types (readonly tuple)")
}

// JS: getShortestPath types > `events` should not require all event types (function)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L80
func TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesFunction(t *testing.T) {
	t.Skip("N/A: type-level only — checked that an `events` function may return a subset of event types")
}

// JS: getShortestPath types > `events` should not allow unknown events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L92
func TestTypes_GetShortestPathTypes_EventsShouldNotAllowUnknownEvents(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on an unknown event type in `events`")
}

// JS: getShortestPath types > `events` should only allow props of a specific event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L108
func TestTypes_GetShortestPathTypes_EventsShouldOnlyAllowPropsOfASpecificEvent(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on a prop belonging to another event type")
}

// JS: getShortestPath types > `serializeEvent` should be allowed to return plain string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L126
func TestTypes_GetShortestPathTypes_SerializeEventShouldBeAllowedToReturnPlainString(t *testing.T) {
	t.Skip("N/A: type-level only — checked that serializeEvent may return a plain (unbranded) string")
}

// JS: getShortestPath types > `serializeState` should be allowed to return plain string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L134
func TestTypes_GetShortestPathTypes_SerializeStateShouldBeAllowedToReturnPlainString(t *testing.T) {
	t.Skip("N/A: type-level only — checked that serializeState may return a plain (unbranded) string")
}

// JS: createTestModel types > `EventExecutor` should be passed event with type that corresponds to its key
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/types.test.ts#L144
func TestTypes_CreateTestModelTypes_EventExecutorShouldBePassedEventWithTypeOfItsKey(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error narrowing of event.type per EventExecutor key")
}
