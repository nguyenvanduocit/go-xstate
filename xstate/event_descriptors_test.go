package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: event descriptors > should fallback to using wildcard transition definition (if specified)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L4
func TestEventDescriptors_ShouldFallbackToUsingWildcardTransitionDefinitionIfSpecified(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"FOO": {{Target: "B"}},
				"*":   {{Target: "C"}},
			}},
			{Key: "B"},
			{Key: "C"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("BAR"))
	assert.Equal(t, "C", service.GetSnapshot().Value)
}

// JS: event descriptors > should prioritize explicit descriptor even if wildcard comes first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L24
func TestEventDescriptors_ShouldPrioritizeExplicitDescriptorEvenIfWildcardComesFirst(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"*":    {{Target: "fail"}},
				"NEXT": {{Target: "pass"}},
			}},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))
	assert.Equal(t, "pass", service.GetSnapshot().Value)
}

// JS: event descriptors > should prioritize explicit descriptor even if a partial one comes first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L44
func TestEventDescriptors_ShouldPrioritizeExplicitDescriptorEvenIfAPartialOneComesFirst(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"foo.*":   {{Target: "fail"}},
				"foo.bar": {{Target: "pass"}},
			}},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("foo.bar"))
	assert.Equal(t, "pass", service.GetSnapshot().Value)
}

// JS: event descriptors > should prioritize a longer descriptor even if the shorter one comes first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L64
func TestEventDescriptors_ShouldPrioritizeALongerDescriptorEvenIfTheShorterOneComesFirst(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"foo.*":     {{Target: "fail"}},
				"foo.bar.*": {{Target: "pass"}},
			}},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("foo.bar.baz"))
	assert.Equal(t, "pass", service.GetSnapshot().Value)
}

// JS: event descriptors > should use a shorter descriptor if the longer one doesn't match
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L84
func TestEventDescriptors_ShouldUseAShorterDescriptorIfTheLongerOneDoesntMatch(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"foo.bar.*": {{
					Target: "fail",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }),
				}},
				"foo.*": {{Target: "pass"}},
			}},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("foo.bar.baz"))
	assert.Equal(t, "pass", service.GetSnapshot().Value)
}

// JS: event descriptors > should fall back to wildcard descriptor when exact descriptor guard fails
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L107
func TestEventDescriptors_ShouldFallBackToWildcardDescriptorWhenExactDescriptorGuardFails(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"foo.bar": {{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }),
					Target: "fail",
				}},
				"foo.*": {{Target: "pass"}},
			}},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("foo.bar"))
	assert.Equal(t, "pass", service.GetSnapshot().Value)
}

// eventDescriptors1StartSuccessMachine mirrors the machine shared (by shape)
// by the wildcard tests: `start` transitions to the final `success` state on
// every given event descriptor.
func eventDescriptors1StartSuccessMachine(descriptors ...string) *xs.StateMachine[any] {
	on := map[string]xs.Transitions{}
	for _, d := range descriptors {
		on[d] = xs.Transitions{{Target: "success"}}
	}
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: on},
			{Key: "success", Type: xs.Final},
		},
	})
}

// JS: event descriptors > should NOT support non-tokenized wildcards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L130
func TestEventDescriptors_ShouldNOTSupportNonTokenizedWildcards(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event*")

	actorRef1 := xs.CreateActor(machine).Start()
	actorRef1.Send(xs.Ev("event"))
	assert.False(t, actorRef1.GetSnapshot().Matches("success"))

	actorRef2 := xs.CreateActor(machine).Start()
	actorRef2.Send(xs.Ev("eventually"))
	assert.False(t, actorRef2.GetSnapshot().Matches("success"))
}

// JS: event descriptors > should support prefix matching with wildcards (+0)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L158
func TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlus0(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event.*")

	actorRef1 := xs.CreateActor(machine).Start()
	actorRef1.Send(xs.Ev("event"))
	assert.True(t, actorRef1.GetSnapshot().Matches("success"))

	actorRef2 := xs.CreateActor(machine).Start()
	actorRef2.Send(xs.Ev("eventually"))
	assert.False(t, actorRef2.GetSnapshot().Matches("success"))
}

// JS: event descriptors > should support prefix matching with wildcards (+1)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L186
func TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlus1(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event.*")

	actorRef1 := xs.CreateActor(machine).Start()
	actorRef1.Send(xs.Ev("event.whatever"))
	assert.True(t, actorRef1.GetSnapshot().Matches("success"))

	actorRef2 := xs.CreateActor(machine).Start()
	actorRef2.Send(xs.Ev("eventually"))
	assert.False(t, actorRef2.GetSnapshot().Matches("success"))

	actorRef3 := xs.CreateActor(machine).Start()
	actorRef3.Send(xs.Ev("eventually.event"))
	assert.False(t, actorRef3.GetSnapshot().Matches("success"))
}

// JS: event descriptors > should support prefix matching with wildcards (+n)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L220
func TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlusN(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event.*")

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("event.first.second"))
	assert.True(t, actorRef.GetSnapshot().Matches("success"))
}

// JS: event descriptors > should support prefix matching with wildcards (+n, multi-prefix)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L242
func TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlusNMultiPrefix(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event.foo.bar.*")

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("event.foo.bar.first.second"))
	assert.True(t, actorRef.GetSnapshot().Matches("success"))
}

// JS: event descriptors > should not match infix wildcards
//
// Warning order follows the iteration order of the `on` keys. Go maps have no
// insertion order, so per docs/docs/porting/core.md ("Ordering differences") the keys
// are visited sorted: "*.event.*" before "event.*.bar.*". JS order is
// "event.*.bar.*" first. Each actor gets its own warn spy (replaces
// warnSpy.mockClear()).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L264
func TestEventDescriptors_ShouldNotMatchInfixWildcards(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event.*.bar.*", "*.event.*")

	warnSpy1 := newSpy()
	actorRef1 := xs.CreateActor(machine, xs.WithWarnHandler(func(args ...any) { warnSpy1.Call(args...) })).Start()
	actorRef1.Send(xs.Ev("event.foo.bar.first.second"))
	assert.False(t, actorRef1.GetSnapshot().Matches("success"))
	assert.Equal(t, [][]any{
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "*.event.*" event.`},
		{`Infix wildcards in transition events are not allowed. Check the "*.event.*" transition.`},
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "event.*.bar.*" event.`},
		{`Infix wildcards in transition events are not allowed. Check the "event.*.bar.*" transition.`},
	}, warnSpy1.Calls())

	warnSpy2 := newSpy()
	actorRef2 := xs.CreateActor(machine, xs.WithWarnHandler(func(args ...any) { warnSpy2.Call(args...) })).Start()
	actorRef2.Send(xs.Ev("whatever.event"))
	assert.False(t, actorRef2.GetSnapshot().Matches("success"))
	assert.Equal(t, [][]any{
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "*.event.*" event.`},
		{`Infix wildcards in transition events are not allowed. Check the "*.event.*" transition.`},
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "event.*.bar.*" event.`},
	}, warnSpy2.Calls())
}

// JS: event descriptors > should not match wildcards as part of tokens
//
// Warning order: `on` keys visited sorted ("*event.*" before "event*.bar.*");
// JS order is "event*.bar.*" first. See docs/docs/porting/core.md "Ordering differences".
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L327
func TestEventDescriptors_ShouldNotMatchWildcardsAsPartOfTokens(t *testing.T) {
	machine := eventDescriptors1StartSuccessMachine("event*.bar.*", "*event.*")

	warnSpy1 := newSpy()
	actorRef1 := xs.CreateActor(machine, xs.WithWarnHandler(func(args ...any) { warnSpy1.Call(args...) })).Start()
	actorRef1.Send(xs.Ev("eventually.bar.baz"))
	assert.False(t, actorRef1.GetSnapshot().Matches("success"))
	assert.Equal(t, [][]any{
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "*event.*" event.`},
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "event*.bar.*" event.`},
	}, warnSpy1.Calls())

	warnSpy2 := newSpy()
	actorRef2 := xs.CreateActor(machine, xs.WithWarnHandler(func(args ...any) { warnSpy2.Call(args...) })).Start()
	actorRef2.Send(xs.Ev("prevent.whatever"))
	assert.False(t, actorRef2.GetSnapshot().Matches("success"))
	assert.Equal(t, [][]any{
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "*event.*" event.`},
		{`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "event*.bar.*" event.`},
	}, warnSpy2.Calls())
}

// JS: event descriptors > should allow assertEvent to use partial descriptors
//
// The FeedbackEvents union and `event.message satisfies string` /
// `event.rate satisfies number` are type-level only.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L381
func TestEventDescriptors_ShouldAllowAssertEventToUsePartialDescriptors(t *testing.T) {
	handleEventSpy := newSpy()
	machine := xs.NewSetup[any](xs.Implementations{
		Actions: map[string]xs.Action{
			"handleEvent": xs.ActionFunc(func(a xs.ActionArgs[any]) {
				xs.AssertEvent(a.Event, "FEEDBACK.*")
				handleEventSpy.Call(a.Event)
			}),
		},
	}).CreateMachine(xs.MachineConfig[any]{
		Initial: "listening",
		States: xs.States{
			{Key: "listening", On: map[string]xs.Transitions{
				"FEEDBACK.*": {{Actions: xs.Actions{xs.ActionRef{Type: "handleEvent"}}}},
			}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.E{"type": "FEEDBACK.MESSAGE", "message": "hello"})
	actor.Send(xs.E{"type": "FEEDBACK.RATE", "rate": 5})

	calls := handleEventSpy.Calls()
	assert.Equal(t, 2, handleEventSpy.Count())
	if assert.Len(t, calls, 2) {
		assert.Equal(t, []any{xs.E{"type": "FEEDBACK.MESSAGE", "message": "hello"}}, calls[0])
		assert.Equal(t, []any{xs.E{"type": "FEEDBACK.RATE", "rate": 5}}, calls[1])
	}
}

// JS: event descriptors > should throw if assertEvent partial descriptor does not match
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/eventDescriptors.test.ts#L439
func TestEventDescriptors_ShouldThrowIfAssertEventPartialDescriptorDoesNotMatch(t *testing.T) {
	nonFeedbackEvent := xs.Ev("OTHER")

	assert.PanicsWithError(t, `Expected event {"type":"OTHER"} to have type matching "FEEDBACK.*"`, func() {
		xs.AssertEvent(nonFeedbackEvent, "FEEDBACK.*")
	})
}
