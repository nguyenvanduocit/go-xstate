package xstate_test

import (
	"context"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every test in types.test.ts L1-1283 is a TypeScript type-level test: none
// calls expect(); each one only checks that a snippet type-checks or that a
// `// @ts-expect-error` line is rejected by the compiler.

// JS: Raise events > should accept a valid event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L40
func TestTypes_RaiseEvents_ShouldAcceptAValidEventType(t *testing.T) {
	t.Skip("N/A: type-level only — raise({type: 'FOO'}) accepted for events FOO|BAR; machine only built, no runtime expectations")
}

// JS: Raise events > should reject an invalid event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L51
func TestTypes_RaiseEvents_ShouldRejectAnInvalidEventType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on raise({type: 'UNKNOWN'}); no runtime expectations")
}

// JS: Raise events > should reject a string event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L63
func TestTypes_RaiseEvents_ShouldRejectAStringEventType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on raise(event) where event.type is string; no runtime expectations")
}

// JS: Raise events > should provide a narrowed down expression event type when used as a transition action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L75
func TestTypes_RaiseEvents_NarrowedExpressionEventTypeInTransitionAction(t *testing.T) {
	t.Skip("N/A: type-level only — event.type narrowed to 'FOO' inside raise(expr) on FOO transition, @ts-expect-error for 'BAR'; no runtime expectations")
}

// JS: Raise events > should accept a valid event type returned from an expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L96
func TestTypes_RaiseEvents_AcceptValidEventTypeReturnedFromExpression(t *testing.T) {
	t.Skip("N/A: type-level only — raise(() => ({type: 'BAR'})) accepted; no runtime expectations")
}

// JS: Raise events > should reject an invalid event type returned from an expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L107
func TestTypes_RaiseEvents_RejectInvalidEventTypeReturnedFromExpression(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on raise(() => ({type: 'UNKNOWN'})); no runtime expectations")
}

// JS: Raise events > should reject a string event type returned from an expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L119
func TestTypes_RaiseEvents_RejectStringEventTypeReturnedFromExpression(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on raise(() => event) where event.type is string; no runtime expectations")
}

// JS: log > should narrow down the event type in the expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L133
func TestTypes_Log_ShouldNarrowDownTheEventTypeInTheExpression(t *testing.T) {
	t.Skip("N/A: type-level only — event.type narrowed to 'FOO' inside log(expr), @ts-expect-error for 'BAR'; no runtime expectations")
}

// JS: stop > should narrow down the event type in the expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L152
func TestTypes_Stop_ShouldNarrowDownTheEventTypeInTheExpression(t *testing.T) {
	t.Skip("N/A: type-level only — event.type narrowed to 'FOO' inside stopChild(expr), @ts-expect-error for 'BAR'; no runtime expectations")
}

// JS: context > defined context in createMachine() should be an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L173
func TestTypes_Context_DefinedContextInCreateMachineShouldBeAnObject(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on context: 'string'; Go's MachineConfig[C] fixes the context type statically; no runtime expectations")
}

// JS: context > context should be required if present in types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L180
func TestTypes_Context_ContextShouldBeRequiredIfPresentInTypes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error when types.context is declared but context is missing; static and lazy context accepted; no runtime expectations")
}

// JS: output > output type should be represented in state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L211
func TestTypes_Output_OutputTypeShouldBeRepresentedInState(t *testing.T) {
	t.Skip("N/A: type-level only — state.output typed number | undefined (@ts-expect-error for number and string); getInitialSnapshot({} as any) only feeds the type check; no runtime expectations")
}

// JS: output > should accept valid static output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L227
func TestTypes_Output_ShouldAcceptValidStaticOutput(t *testing.T) {
	t.Skip("N/A: type-level only — output: 42 accepted for types.output number; no runtime expectations")
}

// JS: output > should reject invalid static output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L236
func TestTypes_Output_ShouldRejectInvalidStaticOutput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on output: 'a string' for types.output number; no runtime expectations")
}

// JS: output > should accept valid dynamic output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L246
func TestTypes_Output_ShouldAcceptValidDynamicOutput(t *testing.T) {
	t.Skip("N/A: type-level only — output: () => 42 accepted for types.output number; no runtime expectations")
}

// JS: output > should reject invalid dynamic output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L255
func TestTypes_Output_ShouldRejectInvalidDynamicOutput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on output: () => 'a string' for types.output number; no runtime expectations")
}

// JS: output > should provide the context type to the dynamic top-level output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L265
func TestTypes_Output_ContextTypeInDynamicTopLevelOutput(t *testing.T) {
	t.Skip("N/A: type-level only — context.password typed string in top-level output fn, @ts-expect-error for number; no runtime expectations")
}

// JS: output > should provide the context type to the dynamic nested output
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L285
func TestTypes_Output_ContextTypeInDynamicNestedOutput(t *testing.T) {
	t.Skip("N/A: type-level only — context.password typed string in nested final-state output fn, @ts-expect-error for number; no runtime expectations")
}

// JS: emitted > emitted type should be represented in actor.on(…)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L321
func TestTypes_Emitted_EmittedTypeShouldBeRepresentedInActorOn(t *testing.T) {
	t.Skip("N/A: type-level only — ev.x satisfies number in actor.on('onClick'), @ts-expect-error on string and on actor.on('unknown'); actor never started; no runtime expectations")
}

// JS: should infer context type from `config.context` when there is no `schema.context`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L346
func TestTypes_ShouldInferContextTypeFromConfigContextWithoutSchemaContext(t *testing.T) {
	t.Skip("N/A: type-level only — context.foo inferred as string in an implementations action, @ts-expect-error for number; no runtime expectations")
}

// JS: should not use actions as possible inference sites
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L365
func TestTypes_ShouldNotUseActionsAsPossibleInferenceSites(t *testing.T) {
	t.Skip("N/A: type-level only — context.count typed number in an implementations action despite inline entry action, @ts-expect-error for string; no runtime expectations")
}

// JS: should work with generic context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L390
func TestTypes_ShouldWorkWithGenericContext(t *testing.T) {
	t.Skip("N/A: type-level only — generic createMachine<TContext> wrapper type-checks; no runtime expectations")
}

// JS: should not widen literal types defined in `schema.context` based on `config.context`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L415
func TestTypes_ShouldNotWidenLiteralSchemaContextTypesBasedOnConfigContext(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on literalTest: 'anything' for 'foo' | 'bar'; no runtime expectations")
}

// JS: states > should accept a state handling subset of events as part of the whole config handling superset of those events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L430
func TestTypes_States_AcceptStateHandlingSubsetOfMachineEvents(t *testing.T) {
	t.Skip("N/A: type-level only — standalone state objects handling a subset of events accepted in states; no runtime expectations")
}

// JS: states > should not accept a state handling an event type outside of the events accepted by the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L461
func TestTypes_States_RejectStateHandlingEventOutsideMachineEvents(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on a state handling TOGGLE_UNDERLINE outside machine events; no runtime expectations")
}

// JS: events > should not use actions as possible inference sites 1
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L484
func TestTypes_Events_ShouldNotUseActionsAsPossibleInferenceSites1(t *testing.T) {
	t.Skip("N/A: type-level only — send({type: 'FOO'}) accepted and @ts-expect-error on send({type: 'UNKNOWN'}) with a raise<any,...> entry; actor start/send has no runtime expectations")
}

// JS: events > should not use actions as possible inference sites 2
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L501
func TestTypes_Events_ShouldNotUseActionsAsPossibleInferenceSites2(t *testing.T) {
	t.Skip("N/A: type-level only — send({type: 'FOO'}) accepted and @ts-expect-error on send({type: 'UNKNOWN'}) with an inline entry; actor start/send has no runtime expectations")
}

// JS: events > event type should be inferable from a simple state machine type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L518
func TestTypes_Events_EventTypeShouldBeInferableFromSimpleStateMachineType(t *testing.T) {
	t.Skip("N/A: type-level only — StateMachine<TContext, TEvent, ...> generic inference; no runtime expectations")
}

// JS: events > should infer inline function parameters when narrowing transition actions based on the event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L558
func TestTypes_Events_InferInlineFuncParamsNarrowingByEventType(t *testing.T) {
	t.Skip("N/A: type-level only — event narrowed to EVENT_WITH_FLAG (event.flag boolean) in inline transition action, @ts-expect-error for 'is not any'; no runtime expectations")
}

// JS: events > should infer inline function parameters when for a wildcard transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L586
func TestTypes_Events_InferInlineFuncParamsForWildcardTransition(t *testing.T) {
	t.Skip("N/A: type-level only — event.type is the full event union in a '*' transition action, @ts-expect-error for 'is not any'; no runtime expectations")
}

// JS: events > should infer inline function parameter with a partial transition descriptor matching multiple events with the matching count of segments
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L615
func TestTypes_Events_InferParamPartialDescriptorMatchingSegmentCount(t *testing.T) {
	t.Skip("N/A: type-level only — 'mouse.click.*' narrows event to mouse.click.up|down (direction 'up'|'down'), @ts-expect-error for 'not any'; no runtime expectations")
}

// JS: events > should infer inline function parameter with a partial transition descriptor matching multiple events with the same count of segments or more
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L640
func TestTypes_Events_InferParamPartialDescriptorSameSegmentCountOrMore(t *testing.T) {
	t.Skip("N/A: type-level only — 'mouse.*' narrows event.type to mouse.click.up|mouse.click.down|mouse.move, @ts-expect-error for 'not any'; no runtime expectations")
}

// JS: events > should not allow a transition using an event type matching the possible prefix but one that is outside of the defines ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L664
func TestTypes_Events_RejectTransitionWithUndefinedEventMatchingPrefix(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on 'mouse.doubleClick' transition key; no runtime expectations")
}

// JS: events > should not allow a transition using an event type matching the possible prefix but one that is outside of the defines ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L681
func TestTypes_Events_RejectTransitionWithUndefinedEventMatchingPrefix2(t *testing.T) {
	t.Skip("N/A: type-level only — duplicate of JS L664 (same name and body): @ts-expect-error on 'mouse.doubleClick' transition key; no runtime expectations")
}

// JS: events > should infer inline function parameter only using a direct match when the transition descriptor doesn't has a trailing wildcard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L698
func TestTypes_Events_InferParamDirectMatchWithoutTrailingWildcard(t *testing.T) {
	t.Skip("N/A: type-level only — 'mouse' key narrows event.type to 'mouse' only, @ts-expect-error for 'not any'; no runtime expectations")
}

// JS: events > should not allow a transition using a partial descriptor related to an event type that is only defined exxactly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L720
func TestTypes_Events_RejectPartialDescriptorForExactlyDefinedEventType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on 'keypress.*' transition key; no runtime expectations")
}

// JS: events > action objects used within implementations parameter should get access to the provided event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L737
func TestTypes_Events_ImplementationActionObjectsGetProvidedEventType(t *testing.T) {
	t.Skip("N/A: type-level only — event.number typed number inside assign in implementations, @ts-expect-error for string; no runtime expectations")
}

// JS: events > should provide the default TEvent to transition actions when there is no specific TEvent configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L763
func TestTypes_Events_DefaultTEventInTransitionActionsWithoutTEvent(t *testing.T) {
	t.Skip("N/A: type-level only — event.type typed string when no types.events configured; no runtime expectations")
}

// JS: events > should provide contextual `event` type in transition actions when the matching event has a union `.type`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L783
func TestTypes_Events_ContextualEventTypeForUnionTypeEvent(t *testing.T) {
	t.Skip("N/A: type-level only — event.type satisfies 'FOO' | 'BAR', event.value satisfies string, @ts-expect-error for number; no runtime expectations")
}

// JS: interpreter > should be convertible to Rx observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L810
func TestTypes_Interpreter_ShouldBeConvertibleToRxObservable(t *testing.T) {
	t.Skip("N/A: type-level only — state.context.count typed number inside rxjs from(actor).subscribe (Symbol.observable interop), @ts-expect-error for string; actor never started; no runtime expectations")
}

// JS: spawnChild action > should reject actor outside of the defined ones at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L832
func TestTypes_SpawnChildAction_RejectActorOutsideDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild('other'); no runtime expectations")
}

// JS: spawnChild action > should accept a defined actor at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L848
func TestTypes_SpawnChildAction_AcceptDefinedActor(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild('child') accepted for types.actors src 'child'; no runtime expectations")
}

// JS: spawnChild action > should allow valid configured actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L862
func TestTypes_SpawnChildAction_AllowValidConfiguredActorID(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild('child', {id: 'ok1'}) accepted for id 'ok1' | 'ok2'; no runtime expectations")
}

// JS: spawnChild action > should disallow invalid actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L877
func TestTypes_SpawnChildAction_DisallowInvalidActorID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild('child', {id: 'child'}) for id 'ok1' | 'ok2'; no runtime expectations")
}

// JS: spawnChild action > should require id to be specified when it was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L898
func TestTypes_SpawnChildAction_RequireIDWhenConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild('child') without id when id is configured; no runtime expectations")
}

// JS: spawnChild action > shouldn't require id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L915
func TestTypes_SpawnChildAction_NotRequireIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild('child') accepted without id; no runtime expectations")
}

// JS: spawnChild action > should allow id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L929
func TestTypes_SpawnChildAction_AllowIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild('child', {id: 'someId'}) accepted; no runtime expectations")
}

// JS: spawnChild action > should allow anonymous inline actor outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L943
func TestTypes_SpawnChildAction_AllowAnonymousInlineActorOutsideConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild(child2) with inline logic accepted; no runtime expectations")
}

// JS: spawnChild action > should disallow anonymous inline actor with an id outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L967
func TestTypes_SpawnChildAction_DisallowAnonymousInlineActorWithConfiguredID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild(child2, {id: 'myChild'}) where 'myChild' is bound to another logic; no runtime expectations")
}

// JS: spawnChild action > should reject static wrong input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L996
func TestTypes_SpawnChildAction_RejectStaticWrongInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on input: 'hello' for number input; no runtime expectations")
}

// JS: spawnChild action > should allow static correct input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1018
func TestTypes_SpawnChildAction_AllowStaticCorrectInput(t *testing.T) {
	t.Skip("N/A: type-level only — input: 42 accepted for number input; no runtime expectations")
}

// JS: spawnChild action > should allow static input that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1036
func TestTypes_SpawnChildAction_AllowStaticInputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — input: 42 accepted for number | string input; no runtime expectations")
}

// JS: spawnChild action > should reject static input that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1054
func TestTypes_SpawnChildAction_RejectStaticInputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on input typed string | number for number input; no runtime expectations")
}

// JS: spawnChild action > should reject dynamic wrong input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1076
func TestTypes_SpawnChildAction_RejectDynamicWrongInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on input: () => 'hello' for number input; no runtime expectations")
}

// JS: spawnChild action > should allow dynamic correct input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1098
func TestTypes_SpawnChildAction_AllowDynamicCorrectInput(t *testing.T) {
	t.Skip("N/A: type-level only — input: () => 42 accepted for number input; no runtime expectations")
}

// JS: spawnChild action > should reject dynamic input that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1116
func TestTypes_SpawnChildAction_RejectDynamicInputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on input: () => number | string for number input; no runtime expectations")
}

// JS: spawnChild action > should allow dynamic input that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1138
func TestTypes_SpawnChildAction_AllowDynamicInputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — input: () => 'hello' accepted for number | string input; no runtime expectations")
}

// JS: spawnChild action > should reject a valid input of a different provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1156
func TestTypes_SpawnChildAction_RejectInputOfDifferentProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild('child1', {input: 'hello'}) where 'hello' fits child2 only; no runtime expectations")
}

// JS: spawnChild action > should require input to be specified when it is required
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1183
func TestTypes_SpawnChildAction_RequireInputWhenRequired(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawnChild('child') without required input; no runtime expectations")
}

// JS: spawnChild action > should not require input when it's optional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1199
func TestTypes_SpawnChildAction_NotRequireInputWhenOptional(t *testing.T) {
	t.Skip("N/A: type-level only — spawnChild('child') accepted for number | undefined input; no runtime expectations")
}

// JS: spawner in assign > spawned actor ref should be compatible with the result of ActorRefFrom
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1217
func TestTypes_SpawnerInAssign_SpawnedRefCompatibleWithActorRefFrom(t *testing.T) {
	t.Skip("N/A: type-level only — Spawner<ProvidedActor> result assignable to ActorRefFrom<...>; spawnChild callback never invoked; no runtime expectations")
}

// JS: spawner in assign > should reject actor outside of the defined ones at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1231
func TestTypes_SpawnerInAssign_RejectActorOutsideDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('other') inside assign; machine never started; no runtime expectations")
}

// JS: spawner in assign > should accept a defined actor at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1249
func TestTypes_SpawnerInAssign_AcceptDefinedActor(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child') accepted inside assign; machine never started; no runtime expectations")
}

// JS: spawner in assign > should allow valid configured actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1266
func TestTypes_SpawnerInAssign_AllowValidConfiguredActorID(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child', {id: 'ok1'}) accepted inside assign; machine never started; no runtime expectations")
}

// types.test.ts L1284-2572. Almost every test here is a TypeScript type-level
// test: it builds machines with `types: {} as {...}` and relies on the
// compiler accepting the snippet or rejecting a `// @ts-expect-error` line.
// Go's contract types `InvokeConfig.Src`/`ID`/`Input`, `SpawnOptions` and
// `Implementations.Actors` loosely (string / any / ActorLogic), so those
// constraints have no Go equivalent. Only tests with runtime expectations are
// ported.

// ---- spawner in assign ----
// In every test of this describe the spawn call sits inside an `entry: assign`
// of a machine that is never started, so nothing runs at runtime.

// JS: spawner in assign > should disallow invalid actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1284
func TestTypes_SpawnerInAssign_DisallowInvalidActorID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('child', {id: 'child'}) when types.actors id is 'ok1'|'ok2'; no runtime expectations")
}

// JS: spawner in assign > should require id to be specified when it was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1305
func TestTypes_SpawnerInAssign_RequireIDWhenConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('child') without id when types.actors configures id; no runtime expectations")
}

// JS: spawner in assign > shouldn't require id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1324
func TestTypes_SpawnerInAssign_NotRequireIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child') without id type-checks when types.actors has no id; no runtime expectations")
}

// JS: spawner in assign > should allow id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1341
func TestTypes_SpawnerInAssign_AllowIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child', {id: 'someId'}) type-checks when types.actors has no id; no runtime expectations")
}

// JS: spawner in assign > should allow anonymous inline actor outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1358
func TestTypes_SpawnerInAssign_AllowAnonymousInlineActorOutsideConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — spawn(child2) with inline logic not in types.actors type-checks; no runtime expectations")
}

// JS: spawner in assign > should no allow anonymous inline actor with an id outside of the configured ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1385
func TestTypes_SpawnerInAssign_DisallowAnonymousInlineActorWithConfiguredID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn(child2, {id: 'myChild'}) when 'myChild' belongs to another configured actor; no runtime expectations")
}

// JS: spawner in assign > should reject static wrong input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1414
func TestTypes_SpawnerInAssign_RejectStaticWrongInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('child', {input: 'hello'}) when child input is number; no runtime expectations")
}

// JS: spawner in assign > should allow static correct input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1436
func TestTypes_SpawnerInAssign_AllowStaticCorrectInput(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child', {input: 42}) type-checks when child input is number; no runtime expectations")
}

// JS: spawner in assign > should allow static input that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1457
func TestTypes_SpawnerInAssign_AllowStaticInputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child', {input: 42}) type-checks when child input is number|string; no runtime expectations")
}

// JS: spawner in assign > should reject static input that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1478
func TestTypes_SpawnerInAssign_RejectStaticInputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn input of type string|number when child input is number; no runtime expectations")
}

// JS: spawner in assign > should reject an attempt to provide dynamic input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1500
func TestTypes_SpawnerInAssign_RejectDynamicInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('child', {input: () => 42}) (spawner input must be static); no runtime expectations")
}

// JS: spawner in assign > should return a concrete actor ref type based on actor logic argument, one that is assignable to a location expecting that concrete actor ref type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1522
func TestTypes_SpawnerInAssign_ConcreteRefFromLogicAssignableToMatchingLocation(t *testing.T) {
	t.Skip("N/A: type-level only — spawn(child) result assignable to context field typed ActorRefFrom<typeof child>; no runtime expectations")
}

// JS: spawner in assign > should return a concrete actor ref type based on actor logic argument, one that isn't assignable to a location expecting a different concrete actor ref type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1549
func TestTypes_SpawnerInAssign_ConcreteRefFromLogicNotAssignableToDifferentLocation(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on assigning spawn(otherChild) to a field typed ActorRefFrom<typeof child>; no runtime expectations")
}

// JS: spawner in assign > should require input to be specified when it is required
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1588
func TestTypes_SpawnerInAssign_RequireInputWhenRequired(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on spawn('child') without input when child input is number; no runtime expectations")
}

// JS: spawner in assign > should not require input when it's optional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1606
func TestTypes_SpawnerInAssign_NotRequireInputWhenOptional(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child') without input type-checks when child input is number|undefined; no runtime expectations")
}

// JS: spawner in assign > should return a concrete actor ref type based on the used string reference
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1625
func TestTypes_SpawnerInAssign_ConcreteRefFromStringReference(t *testing.T) {
	t.Skip("N/A: type-level only — spawn('child') result typed by the matching types.actors entry and assignable to ActorRefFrom<typeof child>; no runtime expectations")
}

// ---- invoke ----

// JS: invoke > should reject actor outside of the defined ones at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1674
func TestTypes_Invoke_RejectActorOutsideDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke {src: 'other'} when types.actors only has 'child'; no runtime expectations")
}

// JS: invoke > should accept a defined actor at usage site
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1691
func TestTypes_Invoke_AcceptDefinedActor(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {src: 'child'} type-checks against types.actors; no runtime expectations")
}

// JS: invoke > should allow valid configured actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1707
func TestTypes_Invoke_AllowValidConfiguredActorID(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {id: 'ok1', src: 'child'} type-checks when id is 'ok1'|'ok2'; no runtime expectations")
}

// JS: invoke > should disallow invalid actor id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1725
func TestTypes_Invoke_DisallowInvalidActorID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke {id: 'child', src: 'child'} when id is 'ok1'|'ok2'; no runtime expectations")
}

// JS: invoke > should require id to be specified when it was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1744
func TestTypes_Invoke_RequireIDWhenConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke {src: 'child'} without id when types.actors configures id; no runtime expectations")
}

// JS: invoke > shouldn't require id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1762
func TestTypes_Invoke_NotRequireIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {src: 'child'} without id type-checks when types.actors has no id; no runtime expectations")
}

// JS: invoke > should allow id to be specified when it was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1778
func TestTypes_Invoke_AllowIDWhenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {id: 'someId', src: 'child'} type-checks when types.actors has no id; no runtime expectations")
}

// JS: invoke > should allow anonymous inline actor outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1795
func TestTypes_Invoke_AllowAnonymousInlineActorOutsideConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {src: child2} with inline logic not in types.actors type-checks; no runtime expectations")
}

// JS: invoke > should diallow anonymous inline actor with an id outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1821
func TestTypes_Invoke_DisallowAnonymousInlineActorWithConfiguredID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke {src: child2, id: 'myChild'} when 'myChild' belongs to another configured actor; no runtime expectations")
}

// JS: invoke > should reject static wrong input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1850
func TestTypes_Invoke_RejectStaticWrongInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke input 'hello' when child input is number; no runtime expectations")
}

// JS: invoke > should allow static correct input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1870
func TestTypes_Invoke_AllowStaticCorrectInput(t *testing.T) {
	t.Skip("N/A: type-level only — invoke input 42 type-checks when child input is number; no runtime expectations")
}

// JS: invoke > should allow static input that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1889
func TestTypes_Invoke_AllowStaticInputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — invoke input 42 type-checks when child input is number|string; no runtime expectations")
}

// JS: invoke > should reject static input that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1908
func TestTypes_Invoke_RejectStaticInputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke input of type string|number when child input is number; no runtime expectations")
}

// JS: invoke > should reject dynamic wrong input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1928
func TestTypes_Invoke_RejectDynamicWrongInput(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke input () => 'hello' when child input is number; no runtime expectations")
}

// JS: invoke > should allow dynamic correct input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1948
func TestTypes_Invoke_AllowDynamicCorrectInput(t *testing.T) {
	t.Skip("N/A: type-level only — invoke input () => 42 type-checks when child input is number; no runtime expectations")
}

// JS: invoke > should reject dynamic input that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1967
func TestTypes_Invoke_RejectDynamicInputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke input () => 42|'hello' when child input is number; no runtime expectations")
}

// JS: invoke > should allow dynamic input that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L1987
func TestTypes_Invoke_AllowDynamicInputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — invoke input () => 'hello' type-checks when child input is number|string; no runtime expectations")
}

// JS: invoke > onDone should work with a service that uses strings for both targets
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2006
func TestTypes_Invoke_OnDoneWithStringsForBothTargets(t *testing.T) {
	// JS onDone: ['.a', '.b']
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				return 1, nil
			}),
			OnDone: xs.Transitions{{Target: ".a"}, {Target: ".b"}},
		}},
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b"},
		},
	})
	assert.NotNil(t, machine) // noop(machine)
	assert.True(t, true)
}

// JS: invoke > onDone should work with a service that uses transition objects for both targets
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2022
func TestTypes_Invoke_OnDoneWithTransitionObjectsForBothTargets(t *testing.T) {
	// JS onDone: [{ target: '.a' }, { target: '.b' }]
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				return 1, nil
			}),
			OnDone: xs.Transitions{{Target: ".a"}, {Target: ".b"}},
		}},
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b"},
		},
	})
	assert.NotNil(t, machine) // noop(machine)
	assert.True(t, true)
}

// JS: invoke > onDone should work with a service that uses a string for one target and a transition object for another
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2038
func TestTypes_Invoke_OnDoneWithStringAndTransitionObjectTargets(t *testing.T) {
	// JS onDone: [{ target: '.a' }, '.b']
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				return 1, nil
			}),
			OnDone: xs.Transitions{{Target: ".a"}, {Target: ".b"}},
		}},
		Initial: "a",
		States: xs.States{
			{Key: "a"},
			{Key: "b"},
		},
	})
	assert.NotNil(t, machine) // noop(machine)
	assert.True(t, true)
}

// JS: invoke > should require input to be specified when it is required
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2054
func TestTypes_Invoke_RequireInputWhenRequired(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on invoke {src: 'child'} without input when child input is number; no runtime expectations")
}

// JS: invoke > should not require input when it's optional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2071
func TestTypes_Invoke_NotRequireInputWhenOptional(t *testing.T) {
	t.Skip("N/A: type-level only — invoke {src: 'child'} without input type-checks when child input is number|undefined; no runtime expectations")
}

// ---- actor implementations ----

// JS: actor implementations > should reject actor outside of the defined ones in provided implementations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2091
func TestTypes_ActorImplementations_RejectActorOutsideDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on implementations.actors.other when types.actors only has 'child'; no runtime expectations")
}

// JS: actor implementations > should accept a defined actor in provided implementations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2112
func TestTypes_ActorImplementations_AcceptDefinedActor(t *testing.T) {
	t.Skip("N/A: type-level only — implementations.actors.child type-checks against types.actors; no runtime expectations")
}

// JS: actor implementations > should reject the provided actor when the output doesn't match
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2132
func TestTypes_ActorImplementations_RejectOutputMismatch(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a promise actor with number output for a string-output slot; no runtime expectations")
}

// JS: actor implementations > should reject the provided actor when its output is a super type of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2153
func TestTypes_ActorImplementations_RejectOutputSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a promise actor with string|number output for a string-output slot; no runtime expectations")
}

// JS: actor implementations > should accept the provided actor when its output is a sub type of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2176
func TestTypes_ActorImplementations_AcceptOutputSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error (JS TODO: ideally accepted) on providing a string-output promise actor for a string|number-output slot; no runtime expectations")
}

// JS: actor implementations > should allow an actor with the expected snapshot type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2200
func TestTypes_ActorImplementations_AllowExpectedSnapshotType(t *testing.T) {
	t.Skip("N/A: type-level only — providing the same machine logic as declared in types.actors type-checks; no runtime expectations")
}

// JS: actor implementations > should reject an actor with an incorrect snapshot type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2229
func TestTypes_ActorImplementations_RejectIncorrectSnapshotType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a machine with context {foo: number} for a {foo: string} slot; no runtime expectations")
}

// JS: actor implementations > should allow an actor with a snapshot type that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2268
func TestTypes_ActorImplementations_AllowSnapshotTypeSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error (JS TODO: ideally allowed) on providing a machine with context {foo: string} for a {foo: string|number} slot; no runtime expectations")
}

// JS: actor implementations > should reject an actor with a snapshot type that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2308
func TestTypes_ActorImplementations_RejectSnapshotTypeSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a machine with context {foo: string|number} for a {foo: string} slot; no runtime expectations")
}

// JS: actor implementations > should allow an actor with the expected event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2347
func TestTypes_ActorImplementations_AllowExpectedEventTypes(t *testing.T) {
	t.Skip("N/A: type-level only — providing the same machine logic (events EV_1) as declared in types.actors type-checks; no runtime expectations")
}

// JS: actor implementations > should reject an actor with wrong event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2373
func TestTypes_ActorImplementations_RejectWrongEventTypes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a machine with events OTHER for an EV_1 slot; no runtime expectations")
}

// JS: actor implementations > should reject an actor with an event type that is a subtype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2406
func TestTypes_ActorImplementations_RejectEventTypeSubtype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on providing a machine with events EV_1 for an EV_1|EV_2 slot; no runtime expectations")
}

// JS: actor implementations > should allow an actor with a snapshot type that is a supertype of the expected one
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2444
func TestTypes_ActorImplementations_AllowEventTypeSupertype(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error (JS TODO: ideally allowed) on providing a machine with events EV_1|EV_2 for an EV_1 slot; no runtime expectations")
}

// ---- state.children without setup ----

// JS: state.children without setup > should return the correct child type on the available snapshot when the child ID for the actor was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2484
func TestTypes_StateChildrenWithoutSetup_CorrectChildTypeWhenChildIDConfigured(t *testing.T) {
	type childCtx struct{ Foo string }

	child := xs.CreateMachine(xs.MachineConfig[childCtx]{
		Context: childCtx{Foo: ""},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{ID: "someChild", Src: "child"}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})

	snapshot := xs.CreateActor(machine).GetSnapshot()
	// JS `snapshot.children.someChild!.getSnapshot()` dereferences the child at runtime.
	childRef := snapshot.Children["someChild"]
	require.NotNil(t, childRef)
	childSnapshot := machineSnap[childCtx](childRef)

	// `childSnapshot.context.foo satisfies string` — enforced by Go's static typing.
	// The `@ts-expect-error` lines (satisfies '' / number | undefined) have no Go equivalent.
	var foo string = childSnapshot.Context.Foo
	_ = foo
}

// JS: state.children without setup > should have an optional child on the available snapshot when the child ID for the actor was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2526
func TestTypes_StateChildrenWithoutSetup_OptionalChildWhenChildIDConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies ActorRefFrom<typeof child> | undefined` and @ts-expect-error on the non-optional form for snapshot.children.myChild; no runtime expectations")
}

// JS: state.children without setup > should have an optional child on the available snapshot when the child ID for the actor was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2550
func TestTypes_StateChildrenWithoutSetup_OptionalChildWhenChildIDNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies ActorRefFrom<typeof child> | undefined` and @ts-expect-error on the non-optional form for snapshot.children.someChild; no runtime expectations")
}

// types.test.ts L2573-3868. No test in this range calls expect(). A test whose
// whole claim is a TypeScript rejection (active `@ts-expect-error` on the
// config itself, or a `types: {}` check with no createMachine/createActor body
// to run) is skipped as N/A-type. Every test whose body builds a machine config
// or actor is ported: the implicit vitest runtime check (nothing throws)
// becomes assert.NotPanics, and in-callback type probes become typed Go
// expressions (`var _ int = a.Context.Count`) that the compiler checks.

// JS: state.children without setup > should not have an index signature on the available snapshot when child IDs were configured for all actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2573
func TestTypes_StateChildrenWithoutSetup_NoIndexSignatureWhenChildIDsConfiguredForAllActors(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that snapshot.children.someChild is rejected when types.actors gives every actor a literal id")
}

// JS: state.children without setup > should have an index signature on the available snapshot when child IDs were configured only for some actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2608
func TestTypes_StateChildrenWithoutSetup_IndexSignatureWhenChildIDsConfiguredOnlyForSomeActors(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies ActorRefFrom<...>` on children.counter/children.someChild and @ts-expect-error that someChild cannot be child1")
}

// JS: state.children with setup and multiple invoke > should type children by their specific actor when using setup with an invoke array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2647
func TestTypes_StateChildrenWithSetupAndMultipleInvoke_TypeChildrenBySpecificActorWithInvokeArray(t *testing.T) {
	type authCtx struct{ Token *string }
	type authOutput struct{ Token string }
	type telemetryCtx struct{ Store *string }
	type consentOutput struct{ Status string }
	type rootCtx struct{ Value int }

	assert.NotPanics(t, func() {
		authLogic := xs.NewSetup[authCtx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"authState": xs.FromPromise(func(context.Context, xs.PromiseArgs) (authOutput, error) {
					return authOutput{Token: "tok"}, nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[authCtx]{
			ID:      "auth",
			Context: authCtx{Token: nil},
			Initial: "idle",
			Invoke: []xs.InvokeConfig{{
				Src: "authState",
				OnDone: xs.Transitions{{Actions: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[authCtx]) authCtx {
						c := a.Context
						tok := a.Event.(xs.DoneActorEvent).Output.(authOutput).Token
						c.Token = &tok
						return c
					}),
				}}},
			}},
			States: xs.States{{Key: "idle"}},
		})

		telemetryLogic := xs.NewSetup[telemetryCtx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"checkConsent": xs.FromPromise(func(context.Context, xs.PromiseArgs) (consentOutput, error) {
					return consentOutput{Status: "granted"}, nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[telemetryCtx]{
			ID:      "telemetry",
			Context: telemetryCtx{Store: nil},
			Initial: "idle",
			States:  xs.States{{Key: "idle"}},
		})

		rootMachine := xs.NewSetup[rootCtx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"auth":      authLogic,
				"telemetry": telemetryLogic,
			},
		}).CreateMachine(xs.MachineConfig[rootCtx]{
			Context: rootCtx{Value: 0},
			Invoke: []xs.InvokeConfig{
				{ID: "auth", SystemID: "auth", Src: "auth"},
				{ID: "telemetry", SystemID: "telemetry", Src: "telemetry"},
			},
		})

		snapshot := xs.CreateActor(rootMachine).GetSnapshot()

		// JS `authChild!.getSnapshot()` throws a TypeError when the child is
		// missing, so existence is part of the runtime claim.
		authChild := snapshot.Children["auth"]
		if assert.NotNil(t, authChild, "children.auth") {
			// `.context.token satisfies string | null`: the Go assignment pins the type.
			var token *string = machineSnap[authCtx](authChild).Context.Token
			assert.Nil(t, token)
		}

		// JS `telemetryChild!.getSnapshot().context.store satisfies string | null`.
		telemetryChild := snapshot.Children["telemetry"]
		if assert.NotNil(t, telemetryChild, "children.telemetry") {
			var store *string = machineSnap[telemetryCtx](telemetryChild).Context.Store
			assert.Nil(t, store)
		}
	})
	// The two `@ts-expect-error ... satisfies number` lines are type-level only.
}

// JS: actions > context should get inferred for builtin actions used as an entry action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2715
func TestTypes_Actions_ContextInferredForBuiltinActionUsedAsEntryAction(t *testing.T) {
	// The JS callbacks (and their type probes) never run: no actor is created.
	// The Go analogue of `((_accept: number) => {})(context.count)` is the
	// typed `var _ int = a.Context.Count`.
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				var _ int = a.Context.Count
				return a.Context // JS `return {}`
			})},
		})
	})
}

// JS: actions > context should get inferred for builtin actions used as a transition action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2732
func TestTypes_Actions_ContextInferredForBuiltinActionUsedAsTransitionAction(t *testing.T) {
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					var _ int = a.Context.Count
					return a.Context
				})}}},
			},
		})
	})
}

// JS: actions > context should get inferred for a builtin action within an array of entry actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2754
func TestTypes_Actions_ContextInferredForBuiltinActionWithinArrayOfEntryActions(t *testing.T) {
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			Entry: xs.Actions{
				xs.ActionRef{Type: "foo"},
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					var _ int = a.Context.Count
					return a.Context
				}),
			},
		})
	})
}

// JS: actions > context should get inferred for a builtin action within an array of transition actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2774
func TestTypes_Actions_ContextInferredForBuiltinActionWithinArrayOfTransitionActions(t *testing.T) {
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{
					xs.ActionRef{Type: "foo"},
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						var _ int = a.Context.Count
						return a.Context
					}),
				}}},
			},
		})
	})
}

// JS: actions > context should get inferred for a stop action used as an entry action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2798
func TestTypes_Actions_ContextInferredForStopActionUsedAsEntryAction(t *testing.T) {
	type ctx struct {
		Count    int
		ChildRef xs.ActorRef
	}
	assert.NotPanics(t, func() {
		childMachine := xs.CreateMachine(xs.MachineConfig[any]{
			Initial: "idle",
			States:  xs.States{{Key: "idle"}},
		})
		xs.CreateMachine(xs.MachineConfig[ctx]{
			ContextFn: func(a xs.ContextArgs) ctx {
				return ctx{Count: 0, ChildRef: a.Spawn(childMachine)}
			},
			Entry: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				var _ int = a.Context.Count
				return a.Context.ChildRef
			}))},
		})
	})
}

// JS: actions > context should get inferred for a stop action used as a transition action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2826
func TestTypes_Actions_ContextInferredForStopActionUsedAsTransitionAction(t *testing.T) {
	type ctx struct {
		Count    int
		ChildRef xs.ActorRef
	}
	assert.NotPanics(t, func() {
		childMachine := xs.CreateMachine(xs.MachineConfig[any]{
			Initial: "idle",
			States:  xs.States{{Key: "idle"}},
		})
		xs.CreateMachine(xs.MachineConfig[ctx]{
			ContextFn: func(a xs.ContextArgs) ctx {
				return ctx{Count: 0, ChildRef: a.Spawn(childMachine)}
			},
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					var _ int = a.Context.Count
					return a.Context.ChildRef
				}))}}},
			},
		})
	})
}

// JS: actions > should report an error when the stop action returns an invalid actor ref
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2858
func TestTypes_Actions_ShouldReportErrorWhenStopActionReturnsInvalidActorRef(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that stopChild(({ context }) => context.count) is rejected because a number is not an actor ref")
}

// JS: actions > context should get inferred for a stop actions within an array of entry actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2877
func TestTypes_Actions_ContextInferredForStopActionsWithinArrayOfEntryActions(t *testing.T) {
	type ctx struct {
		Count      int
		ChildRef   xs.ActorRef
		PromiseRef xs.ActorRef
	}
	assert.NotPanics(t, func() {
		childMachine := xs.CreateMachine(xs.MachineConfig[any]{})
		xs.CreateMachine(xs.MachineConfig[ctx]{
			ContextFn: func(a xs.ContextArgs) ctx {
				return ctx{
					Count:    0,
					ChildRef: a.Spawn(childMachine),
					PromiseRef: a.Spawn(xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
						return "foo", nil
					})),
				}
			},
			Entry: xs.Actions{
				xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					var _ int = a.Context.Count
					return a.Context.ChildRef
				})),
				xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					var _ int = a.Context.Count
					return a.Context.PromiseRef
				})),
			},
		})
	})
}

// JS: actions > should accept assign with partial static object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2910
func TestTypes_Actions_ShouldAcceptAssignWithPartialStaticObject(t *testing.T) {
	type ctx struct {
		Count int
		Mode  *string // 'foo' | 'bar' | null
	}
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0, Mode: nil},
			// assign({ mode: 'foo' })
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				mode := "foo"
				c.Mode = &mode
				return c
			})},
		})
	})
}

// JS: actions > should provide context to single prop updater in assign when it's mixed with a static value for another prop
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2929
func TestTypes_Actions_ContextToSinglePropUpdaterInAssignMixedWithStaticValue(t *testing.T) {
	type ctx struct {
		Count int
		Skip  bool
	}
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0, Skip: true},
			// assign({ count: ({ context }) => context.count + 1, skip: true })
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Count = a.Context.Count + 1
				c.Skip = true
				return c
			})},
		})
	})
}

// JS: actions > should allow a defined parameterized action with params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2951
func TestTypes_Actions_ShouldAllowDefinedParameterizedActionWithParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "greet", Params: map[string]any{"name": "David"}}},
		})
	})
}

// JS: actions > should disallow a non-defined parameterized action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2965
func TestTypes_Actions_ShouldDisallowNonDefinedParameterizedAction(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry { type: 'other' } is rejected when types.actions is greet | poke")
}

// JS: actions > should disallow a defined parameterized action with invalid params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2980
func TestTypes_Actions_ShouldDisallowDefinedParameterizedActionWithInvalidParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that greet params { kick: 'start' } are rejected (expects { name: string })")
}

// JS: actions > should disallow a defined parameterized action when it lacks required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L2995
func TestTypes_Actions_ShouldDisallowDefinedParameterizedActionLackingRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that greet with params {} is rejected (name is required)")
}

// JS: actions > should disallow a defined parameterized action with required params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3008
func TestTypes_Actions_ShouldDisallowParameterizedActionWithRequiredParamsReferencedByString(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: 'greet' is rejected because greet requires params")
}

// JS: actions > should allow a defined action when it has no params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3018
func TestTypes_Actions_ShouldAllowDefinedActionWithNoParamsReferencedByString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "poke"}},
		})
	})
}

// JS: actions > should allow a defined action when it has no params when it's referenced using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3027
func TestTypes_Actions_ShouldAllowDefinedActionWithNoParamsReferencedByObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "poke"}},
		})
	})
}

// JS: actions > should allow a defined action without params when it only has optional params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3038
func TestTypes_Actions_ShouldAllowActionWithOnlyOptionalParamsReferencedByString(t *testing.T) {
	// The JS body references the action with an object ({ type: 'poke' }),
	// not a string, despite the title; mirrored verbatim.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "poke"}},
		})
	})
}

// JS: actions > should allow a defined action without params when it only has optional params when it's referenced using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3051
func TestTypes_Actions_ShouldAllowActionWithOnlyOptionalParamsReferencedByObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "poke"}},
		})
	})
}

// JS: actions > should type action params as undefined in inline custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3064
func TestTypes_Actions_ShouldTypeActionParamsAsUndefinedInInlineCustomAction(t *testing.T) {
	// JS types params as undefined; Go's ActionArgs.Params is `any`, so the
	// type probe reduces to reading the field.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				_ = a.Params
			})},
		})
	})
}

// JS: actions > should type action params as undefined in inline builtin action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3077
func TestTypes_Actions_ShouldTypeActionParamsAsUndefinedInInlineBuiltinAction(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
				_ = a.Params
				return a.Context
			})},
		})
	})
}

// JS: actions > should type action params as the specific defined params in the provided custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3091
func TestTypes_Actions_ShouldTypeActionParamsAsDefinedParamsInProvidedCustomAction(t *testing.T) {
	type greetParams struct{ Name string }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"greet": xs.ActionFunc(func(a xs.ActionArgs[any]) {
					var _ string = a.Params.(greetParams).Name
				}),
			},
		})
	})
}

// JS: actions > should type action params as the specific defined params in the provided builtin action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3112
func TestTypes_Actions_ShouldTypeActionParamsAsDefinedParamsInProvidedBuiltinAction(t *testing.T) {
	type greetParams struct{ Name string }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"greet": xs.Assign(func(a xs.AssignArgs[any]) any {
					var _ string = a.Params.(greetParams).Name
					return a.Context
				}),
			},
		})
	})
}

// JS: actions > should not allow a provided action outside of the defined ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3134
func TestTypes_Actions_ShouldNotAllowProvidedActionOutsideOfDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that implementations.actions.other is rejected when types.actions is greet | poke")
}

// JS: actions > should allow dynamic params that return correct params type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3152
func TestTypes_Actions_ShouldAllowDynamicParamsReturningCorrectParamsType(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{
				Type: "greet",
				Params: xs.NewExpr(func(xs.ExprArgs[any]) any {
					return map[string]any{"name": "Anders"}
				}),
			}},
		})
	})
}

// JS: actions > should disallow dynamic params that return invalid params type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3166
func TestTypes_Actions_ShouldDisallowDynamicParamsReturningInvalidParamsType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that dynamic params returning { surname: 100 } are rejected (expects { surname: string })")
}

// JS: actions > should provide context type to dynamic params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3183
func TestTypes_Actions_ShouldProvideContextTypeToDynamicParams(t *testing.T) {
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 1},
			Entry: xs.Actions{xs.ActionRef{
				Type: "greet",
				Params: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					var _ int = a.Context.Count
					return map[string]any{"name": "Anders"}
				}),
			}},
		})
	})
}

// JS: actions > should provide narrowed down event type to dynamic params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3206
func TestTypes_Actions_ShouldProvideNarrowedDownEventTypeToDynamicParams(t *testing.T) {
	// JS narrows event.type to 'FOO'; Go events are not a union, so the probe
	// reduces to reading the event type as a string.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionRef{
					Type: "greet",
					Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
						var _ string = a.Event.EventType()
						return map[string]any{"name": "Anders"}
					}),
				}}}},
			},
		})
	})
}

// JS: enqueueActions > should be able to enqueue a defined parameterized action with required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3232
func TestTypes_EnqueueActions_ShouldEnqueueDefinedParameterizedActionWithRequiredParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.ActionRef{Type: "greet", Params: map[string]any{"name": "Anders"}})
			})},
		})
	})
}

// JS: enqueueActions > should not allow to enqueue a defined parameterized action without all of its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3248
func TestTypes_EnqueueActions_ShouldNotEnqueueParameterizedActionWithoutRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that enqueue({ type: 'greet', params: {} }) is rejected (name is required)")
}

// JS: enqueueActions > should not be possible to enqueue a parameterized action outside of the defined ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3263
func TestTypes_EnqueueActions_ShouldNotEnqueueParameterizedActionOutsideOfDefinedOnes(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that enqueue({ type: 'other' }) is rejected when types.actions is greet | poke")
}

// JS: enqueueActions > should be possible to enqueue a parameterized action with no required params using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3279
func TestTypes_EnqueueActions_ShouldEnqueueActionWithNoRequiredParamsUsingString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.ActionRef{Type: "poke"})
			})},
		})
	})
}

// JS: enqueueActions > should be possible to enqueue a parameterized action with no required params using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3290
func TestTypes_EnqueueActions_ShouldEnqueueActionWithNoRequiredParamsUsingObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.ActionRef{Type: "poke"})
			})},
		})
	})
}

// JS: enqueueActions > should be able to enqueue an inline custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3301
func TestTypes_EnqueueActions_ShouldEnqueueInlineCustomAction(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"foo": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Enqueue(xs.ActionFunc(func(xs.ActionArgs[any]) {}))
				}),
			},
		})
	})
}

// JS: enqueueActions > should allow a defined simple guard to be checked
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3318
func TestTypes_EnqueueActions_ShouldAllowDefinedSimpleGuardToBeChecked(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"foo": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Check(xs.GuardRef{Type: "plainGuard"})
				}),
			},
		})
	})
}

// JS: enqueueActions > should allow a defined parameterized guard to be checked
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3342
func TestTypes_EnqueueActions_ShouldAllowDefinedParameterizedGuardToBeChecked(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"foo": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Check(xs.GuardRef{Type: "isGreaterThan", Params: map[string]any{"count": 10}})
				}),
			},
		})
	})
}

// JS: enqueueActions > should not allow a guard outside of the defined ones to be checked
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3371
func TestTypes_EnqueueActions_ShouldNotAllowGuardOutsideOfDefinedOnesToBeChecked(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that check('other') is rejected when types.guards is isGreaterThan | plainGuard")
}

// JS: enqueueActions > should type guard params as undefined in inline custom guard when enqueueActions is used in the config
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3398
func TestTypes_EnqueueActions_GuardParamsUndefinedInInlineCustomGuardInConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Check(xs.GuardFunc(func(g xs.GuardArgs[any]) bool {
					_ = g.Params
					return true
				}))
			})},
		})
	})
}

// JS: enqueueActions > should type guard params as undefined in inline custom guard when enqueueActions is used in the implementations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3423
func TestTypes_EnqueueActions_GuardParamsUndefinedInInlineCustomGuardInImplementations(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Actions: map[string]xs.Action{
				"someGuard": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Check(xs.GuardFunc(func(g xs.GuardArgs[any]) bool {
						_ = g.Params
						return true
					}))
				}),
			},
		})
	})
}

// JS: enqueueActions > should be able to enqueue `raise` using its own action creator in a transition with one of the other accepted event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3454
func TestTypes_EnqueueActions_ShouldEnqueueRaiseUsingOwnActionCreator(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"SOMETHING": {{Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
				})}}},
			},
		})
	})
}

// JS: enqueueActions > should be able to enqueue `raise` using its bound action creator in a transition with one of the other accepted event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3475
func TestTypes_EnqueueActions_ShouldEnqueueRaiseUsingBoundActionCreator(t *testing.T) {
	// The contract has no bound creators on EnqueueArgs; `enqueue.raise(ev)`
	// is `enqueue(raise(ev))` (same mapping as actions_3.md).
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"SOMETHING": {{Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
				})}}},
			},
		})
	})
}

// JS: enqueueActions > should not be able to enqueue `raise` using its own action creator in a transition with an event type that is not defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3496
func TestTypes_EnqueueActions_ShouldNotEnqueueRaiseOwnCreatorWithUndefinedEventType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that enqueue(raise({ type: 'OTHER' })) is rejected when types.events is SOMETHING | SOMETHING_ELSE")
}

// JS: enqueueActions > should not be able to enqueue `raise` using its bound action creator in a transition with an event type that is not defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3522
func TestTypes_EnqueueActions_ShouldNotEnqueueRaiseBoundCreatorWithUndefinedEventType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that enqueue.raise({ type: 'OTHER' }) is rejected when types.events is SOMETHING | SOMETHING_ELSE")
}

// JS: input > should provide the input type to the context factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3548
func TestTypes_Input_ShouldProvideInputTypeToContextFactory(t *testing.T) {
	type input struct{ Count int }
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			ContextFn: func(a xs.ContextArgs) any {
				in, _ := a.Input.(input)
				var _ int = in.Count
				return map[string]any{}
			},
		})
	})
}

// JS: input > should accept valid input type when interpreting an actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3564
func TestTypes_Input_ShouldAcceptValidInputTypeWhenInterpretingActor(t *testing.T) {
	type input struct{ Count int }
	assert.NotPanics(t, func() {
		machine := xs.CreateMachine(xs.MachineConfig[any]{})
		xs.CreateActor(machine, xs.WithInput(input{Count: 100}))
	})
}

// JS: input > should reject invalid input type when interpreting an actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3576
func TestTypes_Input_ShouldRejectInvalidInputTypeWhenInterpretingActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that input { count: '' } is rejected when types.input is { count: number }")
}

// JS: input > should require input to be specified when defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3593
func TestTypes_Input_ShouldRequireInputToBeSpecifiedWhenDefined(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that createActor(machine) without input is rejected when types.input is defined")
}

// JS: input > should not require input when not defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3606
func TestTypes_Input_ShouldNotRequireInputWhenNotDefined(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.CreateMachine(xs.MachineConfig[any]{})
		xs.CreateActor(machine)
	})
}

// JS: guards > `not` guard should be accepted when it references another guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3616
func TestTypes_Guards_NotGuardAcceptedWhenReferencingAnotherGuardUsingString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			ID: "b",
			On: map[string]xs.Transitions{
				"EVENT": {{Target: "#b", Guard: xs.Not(xs.GuardRef{Type: "falsy"})}},
			},
		}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"falsy": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
			},
		})
	})
}

// JS: guards > should allow a defined parameterized guard with params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3638
func TestTypes_Guards_ShouldAllowDefinedParameterizedGuardWithParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardRef{Type: "isGreaterThan", Params: map[string]any{"count": 10}}}},
			},
		})
	})
}

// JS: guards > should disallow a non-defined parameterized guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3663
func TestTypes_Guards_ShouldDisallowNonDefinedParameterizedGuard(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that guard { type: 'other' } is rejected when types.guards is isGreaterThan | plainGuard")
}

// JS: guards > should disallow a defined parameterized guard with invalid params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3689
func TestTypes_Guards_ShouldDisallowDefinedParameterizedGuardWithInvalidParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that isGreaterThan params { count: 'bar' } are rejected (expects number)")
}

// JS: guards > should disallow a defined parameterized guard when it lacks required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3715
func TestTypes_Guards_ShouldDisallowDefinedParameterizedGuardLackingRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that isGreaterThan with params {} is rejected (count is required)")
}

// JS: guards > should disallow a defined parameterized guard with required params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3739
func TestTypes_Guards_ShouldDisallowParameterizedGuardWithRequiredParamsReferencedByString(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that guard: 'isGreaterThan' is rejected because it requires params")
}

// JS: guards > should allow a defined guard when it has no params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3760
func TestTypes_Guards_ShouldAllowDefinedGuardWithNoParamsReferencedByString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardRef{Type: "plainGuard"}}},
			},
		})
	})
}

// JS: guards > should allow a defined guard when it has no params when it's referenced using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3780
func TestTypes_Guards_ShouldAllowDefinedGuardWithNoParamsReferencedByObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardRef{Type: "plainGuard"}}},
			},
		})
	})
}

// JS: guards > should allow a defined guard without params when it only has optional params when it's referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3802
func TestTypes_Guards_ShouldAllowGuardWithOnlyOptionalParamsReferencedByString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardRef{Type: "plainGuard"}}},
			},
		})
	})
}

// JS: guards > should allow a defined guard without params when it only has optional params when it's referenced using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3822
func TestTypes_Guards_ShouldAllowGuardWithOnlyOptionalParamsReferencedByObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardRef{Type: "plainGuard"}}},
			},
		})
	})
}

// JS: guards > should type guard params as undefined in inline custom guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3844
func TestTypes_Guards_ShouldTypeGuardParamsAsUndefinedInInlineCustomGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
					_ = a.Params
					return true
				})}},
			},
		})
	})
}

// types.test.ts L3869-4810. No test in this range asserts runtime behaviour with
// expect() except JS L4722, whose expectation is over type-derived `true`
// literals. Convention: every JS test whose body builds a machine/actor or calls
// a method at runtime (including calls under @ts-expect-error / @ts-ignore, which
// still execute in vitest) is ported as assert.NotPanics around the Go
// equivalent; only the type assertions are dropped. A test is N/A-type only when
// Go cannot express the executed call (fromCallback returning a promise / async /
// non-function; createEmptyActor, not in the contract) or when the body contains
// nothing but type-level declarations (IsAny<...>).

// types4Counter and types4Count mirror the JS contexts `{ counter: 0 }` and `{ count: n }`.
type types4Counter struct{ Counter int }

type types4Count struct{ Count int }

// types4IsGreaterThanParams mirrors the JS guard params type `{ count: number }`.
type types4IsGreaterThanParams struct{ Count int }

// JS: guards > should type guard param as unknown in inline composite guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3869
func TestTypes_Guards_TypeGuardParamAsUnknownInInlineCompositeGuard(t *testing.T) {
	// Dropped: `params satisfies unknown` and the two @ts-expect-error checks.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[types4Counter]{
			Context: types4Counter{Counter: 0},
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.Not(xs.GuardFunc(func(a xs.GuardArgs[types4Counter]) bool {
					_ = a.Params
					return true
				}))}},
			},
		})
	})
}

// JS: guards > should type guard params as the specific params in the provided custom guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3899
func TestTypes_Guards_TypeGuardParamsAsSpecificParamsInProvidedCustomGuard(t *testing.T) {
	// Dropped: the compile-time `{count: number}` typing of params and the @ts-expect-error.
	// The guard body is never executed (machine is not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"isGreaterThan": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
					_ = a.Params.(types4IsGreaterThanParams).Count
					return true
				}),
			},
		})
	})
}

// JS: guards > should not type guard params as the specific params in the provided composite guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3926
func TestTypes_Guards_NotTypeGuardParamsAsSpecificParamsInProvidedCompositeGuard(t *testing.T) {
	// Dropped: `params satisfies unknown` and the two @ts-expect-error checks.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[types4Count]{
			Context: types4Count{Count: 0},
		}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"isGreaterThan": xs.Not(xs.GuardFunc(func(a xs.GuardArgs[types4Count]) bool {
					_ = a.Params
					return true
				})),
			},
		})
	})
}

// JS: guards > should not allow a provided guard outside of the defined ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3958
func TestTypes_Guards_NotAllowProvidedGuardOutsideDefinedOnes(t *testing.T) {
	// Dropped: the @ts-expect-error on the key. Go Guards is map[string]Guard, so the
	// same implementations object is accepted; JS still builds the machine at runtime.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"other": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		})
	})
}

// JS: guards > `not` should be allowed in the config argument when inline function gets passed to it
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L3981
func TestTypes_Guards_NotAllowedInConfigArgumentWithInlineFunction(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool {
					return true
				}))}},
			},
		})
	})
}

// JS: guards > `not` should be allowed in the implementations argument when inline function gets passed to it
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4003
func TestTypes_Guards_NotAllowedInImplementationsArgumentWithInlineFunction(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"isGreaterThan": xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool {
					return true
				})),
			},
		})
	})
}

// JS: guards > `stateIn` should be allowed in the config argument
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4027
func TestTypes_Guards_StateInAllowedInConfigArgument(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"EV": {{Guard: xs.StateIn("foo")}},
			},
		})
	})
}

// JS: guards > `stateIn` should be allowed in the implementations argument
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4047
func TestTypes_Guards_StateInAllowedInImplementationsArgument(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{}, xs.Implementations{
			Guards: map[string]xs.Guard{
				"plainGuard": xs.StateIn("foo"),
			},
		})
	})
}

// JS: guards > should allow dynamic params that return correct params type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4069
func TestTypes_Guards_AllowDynamicParamsReturningCorrectParamsType(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Guard: xs.GuardRef{
					Type: "isGreaterThan",
					Params: xs.NewExpr(func(xs.ExprArgs[any]) any {
						return types4IsGreaterThanParams{Count: 100}
					}),
				}}},
			},
		})
	})
}

// JS: guards > should disallow dynamic params that return invalid params type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4092
func TestTypes_Guards_DisallowDynamicParamsReturningInvalidParamsType(t *testing.T) {
	// Dropped: the @ts-expect-error. Go GuardRef.Params is any, so the same config
	// (params returning {count: 'bazinga'}) is built at runtime as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Guard: xs.GuardRef{
					Type: "isGreaterThan",
					Params: xs.NewExpr(func(xs.ExprArgs[any]) any {
						return struct{ Count string }{Count: "bazinga"}
					}),
				}}},
			},
		})
	})
}

// JS: guards > should provide context type to dynamic params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4116
func TestTypes_Guards_ProvideContextTypeToDynamicParams(t *testing.T) {
	// Dropped: `(_accept: number) => {}(context.count)` and the @ts-expect-error
	// (Go ExprArgs[C].Context is statically C). The params fn is never executed.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[types4Count]{
			Context: types4Count{Count: 1},
			On: map[string]xs.Transitions{
				"FOO": {{Guard: xs.GuardRef{
					Type: "isGreaterThan",
					Params: xs.NewExpr(func(a xs.ExprArgs[types4Count]) any {
						return types4IsGreaterThanParams{Count: a.Context.Count}
					}),
				}}},
			},
		})
	})
}

// JS: guards > should provide narrowed down event type to dynamic params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4150
func TestTypes_Guards_ProvideNarrowedDownEventTypeToDynamicParams(t *testing.T) {
	// Dropped: the narrowing of event.type to 'FOO' and the @ts-expect-error.
	// The params fn is never executed.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Guard: xs.GuardRef{
					Type: "isGreaterThan",
					Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
						_ = a.Event.EventType()
						return types4IsGreaterThanParams{Count: 100}
					}),
				}}},
			},
		})
	})
}

// JS: delays > should accept a plain number as key of an after transitions object when delays are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4183
func TestTypes_Delays_AcceptPlainNumberAsAfterKeyWhenDelaysDeclared(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			After: map[string]xs.Transitions{"100": {{}}},
		})
	})
}

// JS: delays > should accept a defined delay type as key of an after transitions object when delays are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4194
func TestTypes_Delays_AcceptDefinedDelayTypeAsAfterKeyWhenDelaysDeclared(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			After: map[string]xs.Transitions{"one second": {{}}},
		})
	})
}

// JS: delays > should reject delay as key of an after transitions object if it's outside of the defined ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4205
func TestTypes_Delays_RejectAfterKeyOutsideDefinedDelays(t *testing.T) {
	// Dropped: the @ts-expect-error. The same config is built at runtime in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			After: map[string]xs.Transitions{"unknown delay": {{}}},
		})
	})
}

// JS: delays > should accept a plain number as delay in `raise` when delays are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4217
func TestTypes_Delays_AcceptPlainNumberDelayInRaiseWhenDelaysDeclared(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: 100 * time.Millisecond})},
		})
	})
}

// JS: delays > should accept a defined delay in `raise`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4226
func TestTypes_Delays_AcceptDefinedDelayInRaise(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: "one minute"})},
		})
	})
}

// JS: delays > should reject a delay outside of the defined ones in `raise`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4235
func TestTypes_Delays_RejectDelayOutsideDefinedOnesInRaise(t *testing.T) {
	// Dropped: the @ts-expect-error. The same config is built at runtime in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: "unknown delay"})},
		})
	})
}

// JS: delays > should accept a plain number as delay in `sendTo` when delays are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4251
func TestTypes_Delays_AcceptPlainNumberDelayInSendToWhenDelaysDeclared(t *testing.T) {
	assert.NotPanics(t, func() {
		otherActor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))

		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.SendTo(otherActor, xs.Ev("FOO"), xs.SendOptions{Delay: 100 * time.Millisecond})},
		})
	})
}

// JS: delays > should accept a defined delay in `sendTo`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4262
func TestTypes_Delays_AcceptDefinedDelayInSendTo(t *testing.T) {
	assert.NotPanics(t, func() {
		otherActor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))

		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.SendTo(otherActor, xs.Ev("FOO"), xs.SendOptions{Delay: "one minute"})},
		})
	})
}

// JS: delays > should reject a delay outside of the defined ones in `sendTo`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4273
func TestTypes_Delays_RejectDelayOutsideDefinedOnesInSendTo(t *testing.T) {
	// Dropped: the @ts-expect-error. The same config is built at runtime in JS.
	assert.NotPanics(t, func() {
		otherActor := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{}))

		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.SendTo(otherActor, xs.Ev("FOO"), xs.SendOptions{Delay: "unknown delay"})},
		})
	})
}

// JS: delays > should accept a plain number as delay in `raise` in `enqueueActions` when delays are declared
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4292
func TestTypes_Delays_AcceptPlainNumberDelayInEnqueueRaiseWhenDelaysDeclared(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: 100 * time.Millisecond}))
			})},
		})
	})
}

// JS: delays > should accept a defined delay in `raise` in `enqueueActions`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4303
func TestTypes_Delays_AcceptDefinedDelayInEnqueueRaise(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: "one minute"}))
			})},
		})
	})
}

// JS: delays > should reject a delay outside of the defined ones in `raise` in `enqueueActions`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4314
func TestTypes_Delays_RejectDelayOutsideDefinedOnesInEnqueueRaise(t *testing.T) {
	// Dropped: the @ts-expect-error. The enqueue callback is never executed (machine not
	// started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: "unknown delay"}))
			})},
		})
	})
}

// JS: delays > should accept any delay string when no explicit delays are defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4331
func TestTypes_Delays_AcceptAnyDelayStringWhenNoExplicitDelaysDefined(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			After: map[string]xs.Transitions{"just_any_delay": {{}}},
		})
	})
}

// JS: tags > should allow a defined tag when it's set using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4341
func TestTypes_Tags_AllowDefinedTagSetUsingString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Tags: xs.Tags{"pending"},
		})
	})
}

// JS: tags > should allow a defined tag when it's set using an array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4350
func TestTypes_Tags_AllowDefinedTagSetUsingArray(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Tags: xs.Tags{"pending"},
		})
	})
}

// JS: tags > should not allow a tag outside of the defined ones when it's set using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4359
func TestTypes_Tags_NotAllowTagOutsideDefinedOnesUsingString(t *testing.T) {
	// Dropped: the @ts-expect-error. Go has only the slice form of Tags.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Tags: xs.Tags{"other"},
		})
	})
}

// JS: tags > should not allow a tag outside of the defined ones when it's set using an array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4369
func TestTypes_Tags_NotAllowTagOutsideDefinedOnesUsingArray(t *testing.T) {
	// Dropped: the @ts-expect-error.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Tags: xs.Tags{"other"},
		})
	})
}

// JS: tags > `hasTag` should allow checking a defined tag
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4381
func TestTypes_Tags_HasTagAllowsCheckingDefinedTag(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.CreateMachine(xs.MachineConfig[any]{})

		actor := xs.CreateActor(machine).Start()

		actor.GetSnapshot().HasTag("a")
	})
}

// JS: tags > `hasTag` should not allow checking a tag outside of the defined ones
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4393
func TestTypes_Tags_HasTagNotAllowCheckingTagOutsideDefinedOnes(t *testing.T) {
	// Dropped: the @ts-expect-error. The call runs at runtime in JS; its result is not asserted.
	assert.NotPanics(t, func() {
		machine := xs.CreateMachine(xs.MachineConfig[any]{})

		actor := xs.CreateActor(machine).Start()

		actor.GetSnapshot().HasTag("other")
	})
}

// JS: fromCallback > should reject a start callback that returns an explicit promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4408
func TestTypes_FromCallback_RejectStartCallbackReturningExplicitPromise(t *testing.T) {
	t.Skip("N/A: type-level only — fromCallback(() => new Promise(...)) under @ts-ignore; Go FromCallback's func(CallbackArgs) func() signature cannot return a promise; no runtime expectations")
}

// JS: fromCallback > should reject a start callback that is an async function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4421
func TestTypes_FromCallback_RejectStartCallbackThatIsAsyncFunction(t *testing.T) {
	t.Skip("N/A: type-level only — fromCallback(async () => {}) under @ts-ignore; Go has no async functions and FromCallback's signature returns func(); no runtime expectations")
}

// JS: fromCallback > should reject a start callback that returns a non-function and non-undefined value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4445
func TestTypes_FromCallback_RejectStartCallbackReturningNonFunctionNonUndefined(t *testing.T) {
	t.Skip("N/A: type-level only — fromCallback(() => 42) under @ts-ignore; Go FromCallback's signature only allows func() or nil; no runtime expectations")
}

// JS: fromCallback > should allow returning an implicit undefined from the start callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4458
func TestTypes_FromCallback_AllowReturningImplicitUndefinedFromStartCallback(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() { return nil }),
			}},
		})
	})
}

// JS: fromCallback > should allow returning an explicit undefined from the start callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4466
func TestTypes_FromCallback_AllowReturningExplicitUndefinedFromStartCallback(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					return nil
				}),
			}},
		})
	})
}

// JS: fromCallback > should allow returning a cleanup function the start callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4476
func TestTypes_FromCallback_AllowReturningCleanupFunctionFromStartCallback(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				// The JS body returns `undefined` despite the test name; ported verbatim.
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					return nil
				}),
			}},
		})
	})
}

// JS: self > should accept correct event types in an inline entry custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4488
func TestTypes_Self_AcceptCorrectEventTypesInInlineEntryCustomAction(t *testing.T) {
	// Dropped: the event-union typing of self.send and the @ts-expect-error for BAZ.
	// The entry action is never executed (machine not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				a.Self.Send(xs.Ev("FOO"))
				a.Self.Send(xs.Ev("BAR"))
				a.Self.Send(xs.Ev("BAZ"))
			})},
		})
	})
}

// JS: self > should accept correct event types in an inline entry builtin action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4502
func TestTypes_Self_AcceptCorrectEventTypesInInlineEntryBuiltinAction(t *testing.T) {
	// Dropped: the event-union typing of self.send and the @ts-expect-error for BAZ.
	// The assign is never executed (machine not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
				a.Self.Send(xs.Ev("FOO"))
				a.Self.Send(xs.Ev("BAR"))
				a.Self.Send(xs.Ev("BAZ"))
				return a.Context
			})},
		})
	})
}

// JS: self > should accept correct event types in an inline transition custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4517
func TestTypes_Self_AcceptCorrectEventTypesInInlineTransitionCustomAction(t *testing.T) {
	// Dropped: the event-union typing of self.send and the @ts-expect-error for BAZ.
	// The action is never executed (machine not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					a.Self.Send(xs.Ev("FOO"))
					a.Self.Send(xs.Ev("BAR"))
					a.Self.Send(xs.Ev("BAZ"))
				})}}},
			},
		})
	})
}

// JS: self > should accept correct event types in an inline transition builtin action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4535
func TestTypes_Self_AcceptCorrectEventTypesInInlineTransitionBuiltinAction(t *testing.T) {
	// Dropped: the event-union typing of self.send and the @ts-expect-error for BAZ.
	// The assign is never executed (machine not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
					a.Self.Send(xs.Ev("FOO"))
					a.Self.Send(xs.Ev("BAR"))
					a.Self.Send(xs.Ev("BAZ"))
					return a.Context
				})}}},
			},
		})
	})
}

// JS: self > should return correct snapshot in an inline entry custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4554
func TestTypes_Self_ReturnCorrectSnapshotInInlineEntryCustomAction(t *testing.T) {
	// Dropped: the `number` vs `string` typing of context.count and the @ts-expect-error
	// (Go Self is the type-erased ActorRef). The entry action is never executed.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[types4Count]{
			Context: types4Count{Count: 0},
			Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[types4Count]) {
				_ = a.Self.AnySnapshot()
			})},
		})
	})
}

// JS: self > should return correct snapshot in an inline entry builtin action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4568
func TestTypes_Self_ReturnCorrectSnapshotInInlineEntryBuiltinAction(t *testing.T) {
	// Dropped: the `number` vs `string` typing of context.count and the @ts-expect-error.
	// The assign is never executed (machine not started), as in JS.
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[types4Count]{
			Context: types4Count{Count: 0},
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[types4Count]) types4Count {
				_ = a.Self.AnySnapshot()
				return a.Context
			})},
		})
	})
}

// JS: createActor > should require input to be specified when it is required
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4585
func TestTypes_CreateActor_RequireInputWhenRequired(t *testing.T) {
	// Dropped: the @ts-expect-error. Go input is an optional ActorOption, so the same
	// createActor(logic) call (without input, never started) runs at runtime as in JS.
	assert.NotPanics(t, func() {
		logic := xs.FromPromise(func(context.Context, xs.PromiseArgs) (int, error) {
			return 100, nil
		})

		xs.CreateActor(logic)
	})
}

// JS: createActor > should not require input when it's optional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4592
func TestTypes_CreateActor_NotRequireInputWhenOptional(t *testing.T) {
	assert.NotPanics(t, func() {
		logic := xs.FromPromise(func(context.Context, xs.PromiseArgs) (int, error) {
			return 100, nil
		})

		xs.CreateActor(logic)
	})
}

// JS: snapshot methods > should type infer actor union snapshot methods
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4602
func TestTypes_SnapshotMethods_TypeInferActorUnionSnapshotMethods(t *testing.T) {
	assert.NotPanics(t, func() {
		// types: events {type: 'one'}, tags 'one'
		typeOne := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "one",
			States:  xs.States{{Key: "one"}},
		})
		_ = typeOne // only used for its type (ActorRefFrom<typeof typeOne>) in JS

		// types: events {type: 'one'} | {type: 'two'}, tags 'one' | 'two'
		typeTwo := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "one",
			States:  xs.States{{Key: "one"}, {Key: "two"}},
		})

		ref := xs.CreateActor(typeTwo)
		snapshot := ref.GetSnapshot()

		snapshot.Can(xs.Ev("one"))
		snapshot.Can(xs.Ev("two"))   // @ts-expect-error in JS
		snapshot.Can(xs.Ev("three")) // @ts-expect-error in JS

		snapshot.HasTag("one")
		snapshot.HasTag("two")   // @ts-expect-error in JS
		snapshot.HasTag("three") // @ts-expect-error in JS

		snapshot.Matches("one")
		snapshot.Matches("two")   // @ts-expect-error in JS
		snapshot.Matches("three") // @ts-expect-error in JS

		snapshot.GetMeta()
		snapshot.ToJSON()
	})
}

// JS: fromPromise should not have issues with actors with emitted types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4657
func TestTypes_FromPromiseNoIssuesWithActorsWithEmittedTypes(t *testing.T) {
	assert.NotPanics(t, func() {
		// types: emitted {type: 'FOO'}
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{})

		actor := xs.CreateActor(machine).Start()

		xs.ToPromise(actor)
	})
}

// JS: UnknownActorRef should return a Snapshot-typed value from getSnapshot()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4669
func TestTypes_UnknownActorRefGetSnapshotReturnsSnapshotTypedValue(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on createEmptyActor().getSnapshot().status === 'FOO' (status union excludes 'FOO'); the comparison result is not asserted")
}

// JS: Actor<T> should be assignable to ActorRefFromLogic<T>
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4676
func TestTypes_ActorShouldBeAssignableToActorRefFromLogic(t *testing.T) {
	type actorThing struct{ actorRef xs.ActorRef }

	assert.NotPanics(t, func() {
		logic := xs.CreateMachine(xs.MachineConfig[any]{})

		newActorThing := func(actorLogic *xs.StateMachine[any]) *actorThing {
			actor := xs.CreateActor(actorLogic)

			// `actor satisfies ActorRefFromLogic<typeof actorLogic>`: the Go
			// compiler checks that *xs.Actor[S] is assignable to xs.ActorRef.
			var ref xs.ActorRef = actor
			return &actorThing{actorRef: ref}
		}

		newActorThing(logic)
	})
}

// JS: AnyStateNode should keep the state nodes of an AnyStateMachine unwidened
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4692
func TestTypes_AnyStateNodeKeepsStateNodesOfAnyStateMachineUnwidened(t *testing.T) {
	t.Skip("N/A: type-level only — IsAny<MetaOf<...>> = true checks that AnyStateNode/AnyStateNodeDefinition meta and transition meta stay `any`; Go has no `any`-tracking type parameters; no runtime expectations")
}

// JS: generic graph and snapshot containers preserve any metadata
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/types.test.ts#L4722
func TestTypes_GenericGraphAndSnapshotContainersPreserveAnyMetadata(t *testing.T) {
	t.Skip("N/A: type-level only — IsAny<...> type annotations over AnyHistoryValue, AnyStateConfig, AnyMachineSnapshot, graph nodes/edges and internal stateUtils return types; the expect(...every(Boolean)).toBe(true) runs over literal `true` constants whose meaning is purely type-level")
}
