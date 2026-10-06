package xstate_test

import (
	"context"
	"math"
	"math/rand"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTypes1UserInput mirrors the JS input type `{ userId: string }`.
type setupTypes1UserInput struct{ UserID string }

// setupTypes1User mirrors the JS output `{ id, name }` of fetchUser.
type setupTypes1User struct {
	ID   string
	Name string
}

// setupTypes1FetchUser mirrors
// `fromPromise(async ({ input }: { input: { userId: string } }) => ({ id: input.userId, name: 'Andarist' }))`.
func setupTypes1FetchUser() *xs.PromiseLogic[setupTypes1User] {
	return xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (setupTypes1User, error) {
		in := a.Input.(setupTypes1UserInput)
		return setupTypes1User{ID: in.UserID, Name: "Andarist"}, nil
	})
}

// setupTypes1FetchAndarist mirrors `fromPromise(async () => ({ name: 'Andarist' }))`.
func setupTypes1FetchAndarist() *xs.PromiseLogic[map[string]any] {
	return xs.FromPromise(func(context.Context, xs.PromiseArgs) (map[string]any, error) {
		return map[string]any{"name": "Andarist"}, nil
	})
}

// JS: setup() > should be able to define a simple function guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L26
func TestSetupTypes_ShouldBeAbleToDefineASimpleFunctionGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		})
	})
}

// JS: setup() > should be able to define a function guard with params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L34
func TestSetupTypes_ShouldBeAbleToDefineAFunctionGuardWithParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				// params: number
				"check": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		})
	})
}

// JS: setup() > should be able to define a function guard that depends on context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L42
func TestSetupTypes_ShouldBeAbleToDefineAFunctionGuardThatDependsOnContext(t *testing.T) {
	type ctx struct{ Enabled bool }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check": xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Enabled }),
			},
		})
	})
}

// JS: setup() > should be able to define a `not` guard referencing another defined simple function guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L53
func TestSetupTypes_NotGuardReferencingDefinedSimpleFunctionGuardUsingString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check":    xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"opposite": xs.Not(xs.GuardRef{Type: "check"}),
			},
		})
	})
}

// JS: setup() > should not accept a `not` guard referencing an unknown guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L62
func TestSetupTypes_ShouldNotAcceptNotGuardReferencingUnknownGuardUsingString(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that not('unknown') rejects a guard name not defined in setup guards")
}

// JS: setup() > should be able to define a `not` guard referencing another defined simple function guard using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L72
func TestSetupTypes_NotGuardReferencingDefinedSimpleFunctionGuardUsingObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check":    xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"opposite": xs.Not(xs.GuardRef{Type: "check"}),
			},
		})
	})
}

// JS: setup() > should not accept a `not` guard referencing an unknown guard using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L83
func TestSetupTypes_ShouldNotAcceptNotGuardReferencingUnknownGuardUsingObject(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that not({ type: 'unknown' }) rejects a guard name not defined in setup guards")
}

// JS: setup() > should be able to define a `not` guard referencing another guard with its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L95
func TestSetupTypes_NotGuardReferencingAnotherGuardWithItsRequiredParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				// params: string
				"check":    xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"opposite": xs.Not(xs.GuardRef{Type: "check", Params: "bar"}),
			},
		})
	})
}

// JS: setup() > should not accept a `not` guard referencing another guard using a string without its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L107
func TestSetupTypes_ShouldNotAcceptNotGuardUsingStringWithoutRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that not('check') is rejected when 'check' requires string params")
}

// JS: setup() > should not accept a `not` guard referencing another guard using an object without its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L117
func TestSetupTypes_ShouldNotAcceptNotGuardUsingObjectWithoutRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that not({ type: 'check' }) is rejected when 'check' requires string params")
}

// JS: setup() > should be able to define a `not` guard referencing another guard with a required mutable array params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L129
func TestSetupTypes_NotGuardReferencingAnotherGuardWithRequiredMutableArrayParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				// params: string[]
				"check":    xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"opposite": xs.Not(xs.GuardRef{Type: "check", Params: []string{"bar", "baz"}}),
			},
		})
	})
}

// JS: setup() > should be able to define a `not` guard that embeds an inline function guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L141
func TestSetupTypes_NotGuardEmbeddingInlineFunctionGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"opposite": xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true })),
			},
		})
	})
}

// JS: setup() > should be able to define a `not` guard that embeds an inline function guard that depends on context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L149
func TestSetupTypes_NotGuardEmbeddingInlineFunctionGuardThatDependsOnContext(t *testing.T) {
	type ctx struct{ Enabled bool }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Guards: map[string]xs.Guard{
				"opposite": xs.Not(xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Enabled })),
			},
		})
	})
}

// JS: setup() > should not be able to define a `not` guard that embeds an inline function guard with params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L160
func TestSetupTypes_ShouldNotDefineNotGuardEmbeddingInlineFunctionGuardWithParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that not() rejects an inline guard declaring typed params")
}

// JS: setup() > should be able to define an `and` guard that references multiple different guards using strings
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L177
func TestSetupTypes_AndGuardReferencingMultipleGuardsUsingStrings(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check1":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(xs.GuardRef{Type: "check1"}, xs.GuardRef{Type: "check2"}),
			},
		})
	})
}

// JS: setup() > should not accept an `and` guard referencing an unknown guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L187
func TestSetupTypes_ShouldNotAcceptAndGuardReferencingUnknownGuardUsingString(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that and(['check1', 'unknown']) rejects a guard name not defined in setup guards")
}

// JS: setup() > should be able to define an `and` guard that references multiple different guards using objects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L198
func TestSetupTypes_AndGuardReferencingMultipleGuardsUsingObjects(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				// check1 params: string; check2 params: number
				"check1": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(
					xs.GuardRef{Type: "check1", Params: "bar"},
					xs.GuardRef{Type: "check2", Params: 42},
				),
			},
		})
	})
}

// JS: setup() > should be able to define an `and` guard that references multiple different guards using both strings and objects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L217
func TestSetupTypes_AndGuardReferencingMultipleGuardsUsingStringsAndObjects(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				// check2 params: number
				"check1": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(
					xs.GuardRef{Type: "check1"},
					xs.GuardRef{Type: "check2", Params: 42},
				),
			},
		})
	})
}

// JS: setup() > should not accept an `and` guard referencing another guard using a string without its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L233
func TestSetupTypes_ShouldNotAcceptAndGuardUsingStringWithoutRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that and(['check1', 'check2']) is rejected when 'check2' requires number params")
}

// JS: setup() > should not accept an `and` guard referencing another guard using an object without its required params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L244
func TestSetupTypes_ShouldNotAcceptAndGuardUsingObjectWithoutRequiredParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that and(['check1', { type: 'check2' }]) is rejected when 'check2' requires number params")
}

// JS: setup() > should be able to define an `and` guard that embeds an inline `not` guard referencing another guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L260
func TestSetupTypes_AndGuardEmbeddingNotGuardReferencingGuardUsingString(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check1":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(xs.GuardRef{Type: "check1"}, xs.Not(xs.GuardRef{Type: "check2"})),
			},
		})
	})
}

// JS: setup() > should not accept an `and` guard that embeds an inline `not` guard referencing an unknown guard using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L270
func TestSetupTypes_ShouldNotAcceptAndGuardEmbeddingNotGuardReferencingUnknownUsingString(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that and(['check1', not('unknown')]) rejects a guard name not defined in setup guards")
}

// JS: setup() > should be able to define an `and` guard that embeds an inline `not` guard referencing another guard using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L281
func TestSetupTypes_AndGuardEmbeddingNotGuardReferencingGuardUsingObject(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check1":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2":        xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(xs.GuardRef{Type: "check1"}, xs.Not(xs.GuardRef{Type: "check2"})),
			},
		})
	})
}

// JS: setup() > should not accept an `and` guard that embeds an inline `not` guard referencing an unknown guard using an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L296
func TestSetupTypes_ShouldNotAcceptAndGuardEmbeddingNotGuardReferencingUnknownUsingObject(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that and(['check1', not({ type: 'unknown' })]) rejects a guard name not defined in setup guards")
}

// JS: setup() > should be able to define an `and` guard that embeds an inline `not` guard embedding an inline function guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L312
func TestSetupTypes_AndGuardEmbeddingNotGuardEmbeddingInlineFunctionGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"check1": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"check2": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				"combinedCheck": xs.And(
					xs.GuardRef{Type: "check1"},
					xs.Not(xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true })),
				),
			},
		})
	})
}

// JS: setup() > should be able to use a parameterized `assign` action with its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L322
func TestSetupTypes_ParameterizedAssignWithRequiredParamsInMachine(t *testing.T) {
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Actions: map[string]xs.Action{
				// params: number
				"resetTo": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Count: a.Params.(int)}
				}),
			},
		}).CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			Entry:   xs.Actions{xs.ActionRef{Type: "resetTo", Params: 0}},
		})
	})
}

// JS: setup() > should not accept a string reference to parameterized `assign` without its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L345
func TestSetupTypes_ShouldNotAcceptStringRefToParameterizedAssignWithoutParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: 'resetTo' is rejected when 'resetTo' requires number params")
}

// JS: setup() > should not accept an object reference to parameterized `assign` without its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L363
func TestSetupTypes_ShouldNotAcceptObjectRefToParameterizedAssignWithoutParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: { type: 'resetTo' } is rejected when 'resetTo' requires number params")
}

// JS: setup() > should not accept an object reference to parameterized `assign` without its required params in the machine
// (second JS test with the same name, JS L383)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L383
func TestSetupTypes_ShouldNotAcceptObjectRefToParameterizedAssignWithoutParams2(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: { type: 'resetTo' } is rejected when 'resetTo' requires number params (duplicate of JS L363)")
}

// JS: setup() > should not accept a reference to parameterized `assign` with wrong params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L403
func TestSetupTypes_ShouldNotAcceptRefToParameterizedAssignWithWrongParams(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: { type: 'resetTo', params: 'foo' } is rejected when 'resetTo' requires number params")
}

// JS: setup() > should not accept a string reference to an unknown action in the machine when actions were configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L424
func TestSetupTypes_ShouldNotAcceptStringRefToUnknownActionWhenActionsConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: 'unknown' is rejected when setup actions are configured")
}

// JS: setup() > should not accept a string reference to an unknown action in the machine when actions were not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L435
func TestSetupTypes_ShouldNotAcceptStringRefToUnknownActionWhenActionsNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: 'unknown' is rejected when setup has no actions")
}

// JS: setup() > should not accept an object reference to an unknown action in the machine when actions were configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L442
func TestSetupTypes_ShouldNotAcceptObjectRefToUnknownActionWhenActionsConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: { type: 'unknown' } is rejected when setup actions are configured")
}

// JS: setup() > should not accept an object reference to an unknown action in the machine when actions were not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L455
func TestSetupTypes_ShouldNotAcceptObjectRefToUnknownActionWhenActionsNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that entry: { type: 'unknown' } is rejected when setup has no actions")
}

// JS: setup() > should accept an `assign` with a spawner that tries to spawn a known actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L464
func TestSetupTypes_ShouldAcceptAssignWithSpawnerSpawningKnownActor(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"fetchUser": setupTypes1FetchAndarist(),
			},
			Actions: map[string]xs.Action{
				"spawnFetcher": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Child = a.Spawn("fetchUser")
					return c
				}),
			},
		})
	})
}

// JS: setup() > should not accept an `assign` with a spawner that tries to spawn an unknown actor when actors are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L479
func TestSetupTypes_ShouldNotAcceptAssignSpawningUnknownActorWhenActorsConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that spawn('unknown') inside assign is rejected when setup actors are configured")
}

// JS: setup() > should not accept an `assign` with a spawner that tries to spawn an unknown actor when actors are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L496
func TestSetupTypes_ShouldNotAcceptAssignSpawningUnknownActorWhenActorsNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that spawn('unknown') inside assign is rejected when setup has no actors")
}

// JS: setup() > should not accept an invoke that tries to invoke an unknown actor when actors are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L510
func TestSetupTypes_ShouldNotAcceptInvokeOfUnknownActorWhenActorsNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke: { src: 'unknown' } is rejected when setup has no actors")
}

// JS: setup() > should not accept a non-logic actor when children were not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L519
func TestSetupTypes_ShouldNotAcceptNonLogicActorWhenChildrenNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that actors: { increment: 'bazinga' } is rejected (in Go, map[string]xs.ActorLogic cannot hold a string)")
}

// JS: setup() > should accept a `spawnChild` action that tries to spawn a known actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L528
func TestSetupTypes_ShouldAcceptSpawnChildActionSpawningKnownActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"fetchUser": setupTypes1FetchAndarist(),
			},
			Actions: map[string]xs.Action{
				"spawnFetcher": xs.SpawnChild("fetchUser"),
			},
		})
	})
}

// JS: setup() > should not accept a `spawnChild` action that tries to spawn an unknown actor when actors are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L539
func TestSetupTypes_ShouldNotAcceptSpawnChildOfUnknownActorWhenActorsConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that spawnChild('unknown') is rejected when setup actors are configured")
}

// JS: setup() > should not accept a `spawnChild` action that tries to spawn an unknown actor when actors are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L553
func TestSetupTypes_ShouldNotAcceptSpawnChildOfUnknownActorWhenActorsNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that spawnChild('unknown') is rejected when setup has no actors")
}

// JS: setup() > should accept a `raise` action that raises a known event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L563
func TestSetupTypes_ShouldAcceptRaiseActionThatRaisesKnownEvent(t *testing.T) {
	// types.events: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"raiseFoo": xs.Raise(xs.Ev("FOO")),
			},
		})
	})
}

// JS: setup() > should not accept a `raise` action that raises an unknown event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L582
func TestSetupTypes_ShouldNotAcceptRaiseActionThatRaisesUnknownEvent(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that raise({ type: 'BAZ' }) is rejected when types.events is FOO | BAR")
}

// JS: setup() > should accept a `raise` action that references a known delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L602
func TestSetupTypes_ShouldAcceptRaiseActionThatReferencesKnownDelay(t *testing.T) {
	// types.events: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"raiseFoo": xs.Raise(xs.Ev("FOO"), xs.SendOptions{Delay: "hundred"}),
			},
			Delays: map[string]any{
				"hundred": ms(100),
			},
		})
	})
}

// JS: setup() > should not accept a `raise` action that references an unknown delay when delays are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L629
func TestSetupTypes_ShouldNotAcceptRaiseWithUnknownDelayWhenDelaysConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that raise(..., { delay: 'hundred' }) is rejected when only 'thousand' is a configured delay")
}

// JS: setup() > should not accept a `raise` action that references an unknown delay when delays are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L657
func TestSetupTypes_ShouldNotAcceptRaiseWithUnknownDelayWhenDelaysNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that raise(..., { delay: 'hundred' }) is rejected when setup has no delays")
}

// JS: setup() > should accept a `sendTo` action that references a known delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L682
func TestSetupTypes_ShouldAcceptSendToActionThatReferencesKnownDelay(t *testing.T) {
	// types.events: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"sendFoo": xs.SendTo(
					xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Self }),
					xs.Ev("FOO"),
					xs.SendOptions{Delay: "hundred"},
				),
			},
			Delays: map[string]any{
				"hundred": ms(100),
			},
		})
	})
}

// JS: setup() > should not accept a `sendTo` action that references an unknown delay when delays are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L710
func TestSetupTypes_ShouldNotAcceptSendToWithUnknownDelayWhenDelaysConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that sendTo(..., { delay: 'hundred' }) is rejected when only 'thousand' is a configured delay")
}

// JS: setup() > should not accept a `sendTo` action that references an unknown delay when delays are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L739
func TestSetupTypes_ShouldNotAcceptSendToWithUnknownDelayWhenDelaysNotConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that sendTo(..., { delay: 'hundred' }) is rejected when setup has no delays")
}

// JS: setup() > should accept a `sendTo` action that send an event to `self` when delays are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L765
func TestSetupTypes_ShouldAcceptSendToSelfWhenDelaysNotConfigured(t *testing.T) {
	// types.events: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"sendFoo": xs.SendTo(
					xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Self }),
					xs.Ev("FOO"),
				),
			},
		})
	})
}

// JS: setup() > should accept a `sendParent` action when delays are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L784
func TestSetupTypes_ShouldAcceptSendParentActionWhenDelaysNotConfigured(t *testing.T) {
	// types.events: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"sendFoo": xs.SendParent(xs.Ev("FOO")),
			},
		})
	})
}

// JS: setup() > should accept an `emit` action that emits a known event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L803
func TestSetupTypes_ShouldAcceptEmitActionThatEmitsKnownEvent(t *testing.T) {
	// types.emitted: { type: 'FOO' } | { type: 'BAR' }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"emitFoo": xs.Emit(xs.Ev("FOO")),
			},
		})
	})
}

// JS: setup() > should not accept an `emit` action that emits an unknown event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L822
func TestSetupTypes_ShouldNotAcceptEmitActionThatEmitsUnknownEvent(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that emit({ type: 'BAZ' }) is rejected when types.emitted is FOO | BAR")
}

// JS: setup() > should be able to use an output of specific actor in the `assign` within `invoke`'s `onDone` in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L842
func TestSetupTypes_ActorOutputInAssignWithinInvokeOnDone(t *testing.T) {
	// The JS `event.output satisfies string` / `@ts-expect-error event.output
	// satisfies number` checks are type-level only; the runtime part
	// (setup + createMachine does not throw) is ported.
	type ctx struct{ Data any }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"greet": xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "hello", nil
				}),
				"throwDice": xs.FromPromise(func(context.Context, xs.PromiseArgs) (float64, error) {
					return rand.Float64(), nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[ctx]{
			Invoke: []xs.InvokeConfig{{
				Src: "greet",
				OnDone: xs.Transitions{{Actions: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Data = map[string]any{}
						return c
					}),
				}}},
			}},
		})
	})
}

// JS: setup() > should be able to use an output of specific actor in the custom action within `invoke`'s `onDone` in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L866
func TestSetupTypes_ActorOutputInCustomActionWithinInvokeOnDone(t *testing.T) {
	// The JS `event.output satisfies string` / `@ts-expect-error event.output
	// satisfies number` checks are type-level only; the runtime part
	// (setup + createMachine does not throw) is ported.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"greet": xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "hello", nil
				}),
				"throwDice": xs.FromPromise(func(context.Context, xs.PromiseArgs) (float64, error) {
					return rand.Float64(), nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Src: "greet",
				OnDone: xs.Transitions{{Actions: xs.Actions{
					xs.ActionFunc(func(xs.ActionArgs[any]) {}),
				}}},
			}},
		})
	})
}

// JS: setup() > should accept a compatible provided logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L887
func TestSetupTypes_ShouldAcceptCompatibleProvidedLogic(t *testing.T) {
	type state struct{ Count int }
	identity := func(s state, _ xs.Event, _ *xs.ActorScope) state { return s }
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"reducer": xs.FromTransition(identity, func(xs.TransitionInitArgs) state { return state{Count: 42} }),
			},
		}).CreateMachine(xs.MachineConfig[any]{}).Provide(xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"reducer": xs.FromTransition(identity, func(xs.TransitionInitArgs) state { return state{Count: 100} }),
			},
		})
	})
}

// JS: setup() > should allow anonymous inline actor outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L901
func TestSetupTypes_ShouldAllowAnonymousInlineActorOutsideConfiguredActors(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"known": xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "known", nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Logic: xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "inline", nil
				}),
			}},
		})
	})
}

// JS: setup() > should disallow anonymous inline actor with an id outside of the configured actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L913
func TestSetupTypes_ShouldDisallowAnonymousInlineActorWithIDOutsideConfiguredActors(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke: { src: <inline logic>, id: 'myChild' } is rejected when setup actors are configured")
}

// JS: setup() > should not accept an incompatible provided logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L927
func TestSetupTypes_ShouldNotAcceptIncompatibleProvidedLogic(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that provide() rejects a fromTransition logic whose state type differs from the configured 'reducer'")
}

// JS: setup() > should allow actors to be defined without children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L942
func TestSetupTypes_ShouldAllowActorsToBeDefinedWithoutChildren(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"foo": xs.CreateMachine(xs.MachineConfig[any]{}),
			},
		})
	})
}

// JS: setup() > should allow actors to be defined with children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L950
func TestSetupTypes_ShouldAllowActorsToBeDefinedWithChildren(t *testing.T) {
	// types.children: { first: 'foo'; second: 'bar' } is type-level only.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"foo": xs.CreateMachine(xs.MachineConfig[any]{}),
				"bar": xs.CreateMachine(xs.MachineConfig[any]{}),
			},
		})
	})
}

// JS: setup() > should not allow actors to be defined without all required children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L965
func TestSetupTypes_ShouldNotAllowActorsWithoutAllRequiredChildren(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that actors missing 'bar' is rejected when types.children requires 'foo' and 'bar'")
}

// JS: setup() > should require actors to be defined when children are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L980
func TestSetupTypes_ShouldRequireActorsWhenChildrenAreConfigured(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that setup() without actors is rejected when types.children is configured")
}

// JS: setup() > should allow more actors to be defined than the ones required by children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L994
func TestSetupTypes_ShouldAllowMoreActorsThanRequiredByChildren(t *testing.T) {
	// types.children: { first: 'foo'; second: 'bar' } is type-level only.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"foo": xs.CreateMachine(xs.MachineConfig[any]{}),
				"bar": xs.CreateMachine(xs.MachineConfig[any]{}),
				"baz": xs.CreateMachine(xs.MachineConfig[any]{}),
			},
		})
	})
}

// JS: setup() > should allow an actor with input to be provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1010
func TestSetupTypes_ShouldAllowActorWithInputToBeProvided(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"fetchUser": setupTypes1FetchUser(),
			},
		})
	})
}

// JS: setup() > should reject static wrong input when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1023
func TestSetupTypes_ShouldRejectStaticWrongInputWhenInvokingProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke input 4157 is rejected for an actor whose input is { userId: string }")
}

// JS: setup() > should allow static correct input when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1042
func TestSetupTypes_ShouldAllowStaticCorrectInputWhenInvokingProvidedActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"fetchUser": setupTypes1FetchUser(),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Src:   "fetchUser",
				Input: setupTypes1UserInput{UserID: "4nd4r157"},
			}},
		})
	})
}

// JS: setup() > should allow static input that is a subtype of the expected one when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1062
func TestSetupTypes_ShouldAllowStaticSubtypeInputWhenInvokingProvidedActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				// input: number | string
				"child": xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "foo", nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Src:   "child",
				Input: 42,
			}},
		})
	})
}

// JS: setup() > should reject static input that is a supertype of the expected one when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1077
func TestSetupTypes_ShouldRejectStaticSupertypeInputWhenInvokingProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke input { userId } | 42 is rejected for an actor whose input is { userId: string }")
}

// JS: setup() > should reject dynamic wrong input when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1101
func TestSetupTypes_ShouldRejectDynamicWrongInputWhenInvokingProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke input () => 42 is rejected for an actor whose input is { userId: string }")
}

// JS: setup() > should allow dynamic correct input when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1120
func TestSetupTypes_ShouldAllowDynamicCorrectInputWhenInvokingProvidedActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"fetchUser": setupTypes1FetchUser(),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Src: "fetchUser",
				Input: xs.NewExpr(func(xs.ExprArgs[any]) any {
					return setupTypes1UserInput{UserID: "4nd4r157"}
				}),
			}},
		})
	})
}

// JS: setup() > should reject dynamic input that is a supertype of the expected one when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1140
func TestSetupTypes_ShouldRejectDynamicSupertypeInputWhenInvokingProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke input () => { userId } | 42 is rejected for an actor whose input is { userId: string }")
}

// JS: setup() > should allow dynamic input that is a subtype of the expected one when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1164
func TestSetupTypes_ShouldAllowDynamicSubtypeInputWhenInvokingProvidedActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				// input: number | string
				"child": xs.FromPromise(func(context.Context, xs.PromiseArgs) (string, error) {
					return "foo", nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{
				Src:   "child",
				Input: xs.NewExpr(func(xs.ExprArgs[any]) any { return "hello" }),
			}},
		})
	})
}

// JS: setup() > should reject a valid input of a different provided actor when invoking a provided actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1179
func TestSetupTypes_ShouldRejectValidInputOfDifferentProvidedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoking 'fetchUser' with rollADie's number input 0.31 is rejected")
}

// JS: setup() > should require input to be specified when it is required by the invoked actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1201
func TestSetupTypes_ShouldRequireInputWhenRequiredByInvokedActor(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that invoke: { src: 'fetchUser' } without input is rejected when the actor requires input")
}

// JS: setup() > should not require input when it's optional in the invoked actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1219
func TestSetupTypes_ShouldNotRequireInputWhenOptionalInInvokedActor(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				// input: number | undefined
				"rollADie": xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (float64, error) {
					if in, _ := a.Input.(float64); in != 0 {
						return math.Min(rand.Float64(), in), nil
					}
					return rand.Float64(), nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Invoke: []xs.InvokeConfig{{Src: "rollADie"}},
		})
	})
}

// JS: setup() > should provide contextual parameters to input factory for an actor that doesn't specify any input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1234
func TestSetupTypes_ShouldProvideContextualParamsToInputFactoryOfActorWithoutInput(t *testing.T) {
	// The JS `@ts-expect-error context.foo` check is type-level only (Go
	// rejects `a.Context.Foo` at compile time); the runtime part is ported.
	type ctx struct{ Count int }
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{
			Actors: map[string]xs.ActorLogic{
				"child": xs.FromPromise(func(context.Context, xs.PromiseArgs) (int, error) {
					return 1, nil
				}),
			},
		}).CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 1},
			Invoke: []xs.InvokeConfig{{
				Src:   "child",
				Input: xs.NewExpr(func(xs.ExprArgs[ctx]) any { return nil }),
			}},
		})
	})
}

// JS: setup() > should return the correct child type on the available snapshot when the child ID for the actor was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1256
func TestSetupTypes_ShouldReturnCorrectChildTypeOnSnapshotWhenChildIDConfigured(t *testing.T) {
	type childCtx struct{ Foo string }

	child := xs.CreateMachine(xs.MachineConfig[childCtx]{
		Context: childCtx{Foo: ""},
	})

	// types.children: { someChild: 'child' } is type-level only.
	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"child": child,
		},
	}).CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{ID: "someChild", Src: "child"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()
	// JS `snapshot.children.someChild!.getSnapshot()` throws if the child is absent.
	require.NotNil(t, snapshot.Children["someChild"])
	childSnapshot := machineSnap[childCtx](snapshot.Children["someChild"])

	// `childSnapshot.context.foo satisfies string | undefined` and
	// `satisfies string`: compile-time check that the field is a string.
	var _ string = childSnapshot.Context.Foo
	// The two `@ts-expect-error` checks (`satisfies ''`, `satisfies number | undefined`)
	// are type-level only and have no Go equivalent.
}

// setup.types.test.ts L1295-2577 has no expect() calls. Every test still runs
// code at runtime (setup/createMachine/createActor/start/matches/getMeta).
// Each Go test keeps that runtime code and drops only the type-level
// statements (`satisfies`, `@ts-expect-error`, EventFrom / ContextFrom casts,
// which Go's compiler enforces or which have no Go equivalent). The implicit
// JS check (nothing throws) becomes assert.NotPanics. JS discards every
// snapshot.value / matches() / children / getMeta() result; where the Go test
// asserts one, it is the value the JS runtime computes (matchesState in
// utils.ts, getValueFromAdj in stateUtils.ts, machineSnapshotGetMeta in
// State.ts) and is marked "beyond JS".

// setupTypes2SimpleFSM is the green/yellow/red machine shared by JS L1780-1833.
func setupTypes2SimpleFSM() *xs.StateMachine[any] {
	return xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green"},
			{Key: "yellow"},
			{Key: "red"},
		},
	})
}

// setupTypes2NestedFSM is the machine with a compound `green` state shared by
// JS L1853-1976.
func setupTypes2NestedFSM() *xs.StateMachine[any] {
	return xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", Initial: "walk", States: xs.States{
				{Key: "walk"},
				{Key: "wait"},
			}},
			{Key: "yellow"},
			{Key: "red"},
		},
	})
}

// JS: setup() > should have an optional child on the available snapshot when the child ID for the actor was configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1295
func TestSetupTypes_Setup_OptionalChildOnSnapshotWhenChildIDConfigured(t *testing.T) {
	type childCtx struct{ Counter int }

	assert.NotPanics(t, func() {
		child := xs.CreateMachine(xs.MachineConfig[childCtx]{Context: childCtx{Counter: 0}})

		machine := xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{"child": child},
		}).CreateMachine(xs.MachineConfig[any]{})

		childActor := xs.CreateActor(machine).GetSnapshot().Children["myChild"]

		// Beyond JS: no child was spawned, so the entry is absent (JS `undefined`).
		assert.Nil(t, childActor)
	})
}

// JS: setup() > should have an optional child on the available snapshot when the child ID for the actor was not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1320
func TestSetupTypes_Setup_OptionalChildOnSnapshotWhenChildIDNotConfigured(t *testing.T) {
	type childCtx struct{ Counter int }

	assert.NotPanics(t, func() {
		child := xs.CreateMachine(xs.MachineConfig[childCtx]{Context: childCtx{Counter: 0}})

		machine := xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{"child": child},
		}).CreateMachine(xs.MachineConfig[any]{})

		childActor := xs.CreateActor(machine).GetSnapshot().Children["someChild"]

		// Beyond JS: no child was spawned, so the entry is absent (JS `undefined`).
		assert.Nil(t, childActor)
	})
}

// JS: setup() > should not have an index signature on the available snapshot when child IDs were configured for all actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1340
func TestSetupTypes_Setup_NoIndexSignatureOnSnapshotWhenAllChildIDsConfigured(t *testing.T) {
	type child1Ctx struct{ Counter int }
	type child2Ctx struct{ Answer string }

	assert.NotPanics(t, func() {
		child1 := xs.CreateMachine(xs.MachineConfig[child1Ctx]{Context: child1Ctx{Counter: 0}})
		child2 := xs.CreateMachine(xs.MachineConfig[child2Ctx]{Context: child2Ctx{Answer: ""}})

		machine := xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{"child1": child1, "child2": child2},
		}).CreateMachine(xs.MachineConfig[any]{})

		// Beyond JS: JS only reads these properties; no child was spawned, so
		// every entry is absent (JS `undefined`).
		assert.Nil(t, xs.CreateActor(machine).GetSnapshot().Children["counter"])
		assert.Nil(t, xs.CreateActor(machine).GetSnapshot().Children["quiz"])
		assert.Nil(t, xs.CreateActor(machine).GetSnapshot().Children["someChild"])
	})
}

// JS: setup() > should have an index signature on the available snapshot when child IDs were configured only for some actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1372
func TestSetupTypes_Setup_IndexSignatureOnSnapshotWhenSomeChildIDsConfigured(t *testing.T) {
	type child1Ctx struct{ Counter int }
	type child2Ctx struct{ Answer string }

	assert.NotPanics(t, func() {
		child1 := xs.CreateMachine(xs.MachineConfig[child1Ctx]{Context: child1Ctx{Counter: 0}})
		child2 := xs.CreateMachine(xs.MachineConfig[child2Ctx]{Context: child2Ctx{Answer: ""}})

		machine := xs.NewSetup[any](xs.Implementations{
			Actors: map[string]xs.ActorLogic{"child1": child1, "child2": child2},
		}).CreateMachine(xs.MachineConfig[any]{})

		// Beyond JS: no child was spawned, so both entries are absent.
		assert.Nil(t, xs.CreateActor(machine).GetSnapshot().Children["counter"])
		assert.Nil(t, xs.CreateActor(machine).GetSnapshot().Children["someChild"])
	})
}

// JS: setup() > should type the snapshot state value of a stateless machine as an empty object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1406
func TestSetupTypes_Setup_StateValueOfStatelessMachineIsEmptyObjectType(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: getValueFromAdj returns {} for a root without states.
		assert.IsType(t, map[string]any{}, snapshot.Value)
		assert.Empty(t, snapshot.Value)
	})
}

// JS: setup() > should type the snapshot state value of a simple FSM as a union of strings
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1420
func TestSetupTypes_Setup_StateValueOfSimpleFSMIsUnionOfStrings(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a"},
				{Key: "b"},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: the value is the initial state key.
		assert.Equal(t, "a", snapshot.Value)
	})
}

// JS: setup() > should type the snapshot state value without including history state keys
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1437
func TestSetupTypes_Setup_StateValueTypeExcludesHistoryStateKeys(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a"},
				{Key: "b"},
				{Key: "c", Type: xs.History},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: the value is the initial state key.
		assert.Equal(t, "a", snapshot.Value)
	})
}

// JS: setup() > should type the snapshot state value of a nested statechart using optional properties for parent states keys
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1457
func TestSetupTypes_Setup_StateValueOfNestedStatechartUsesOptionalParentKeys(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", Initial: "a1", States: xs.States{
					{Key: "a1"},
					{Key: "a2"},
				}},
				{Key: "b", Initial: "b1", States: xs.States{
					{Key: "b1"},
					{Key: "b2"},
				}},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: only the active parent key is present.
		assert.Equal(t, map[string]any{"a": "a1"}, snapshot.Value)
	})
}

// JS: setup() > should type the snapshot state value of a parallel state using required properties for its children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1492
func TestSetupTypes_Setup_StateValueOfParallelStateUsesRequiredChildKeys(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Type: xs.Parallel,
			States: xs.States{
				{Key: "a", Initial: "a1", States: xs.States{
					{Key: "a1"},
					{Key: "a2"},
				}},
				{Key: "b", Initial: "b1", States: xs.States{
					{Key: "b1"},
					{Key: "b2"},
				}},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: every region is present.
		assert.Equal(t, map[string]any{"a": "a1", "b": "b1"}, snapshot.Value)
	})
}

// JS: setup() > should type the snapshot state value of an empty parallel region as an empty object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1524
func TestSetupTypes_Setup_StateValueOfEmptyParallelRegionIsEmptyObjectType(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Type: xs.Parallel,
			States: xs.States{
				{Key: "a"},
				{Key: "b", Initial: "b1", States: xs.States{
					{Key: "b1"},
					{Key: "b2"},
				}},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: the atomic region `a` has the value {} (getValueFromAdj).
		valueMap, ok := snapshot.Value.(map[string]any)
		if assert.True(t, ok, "value is a map") {
			assert.Equal(t, "b1", valueMap["b"])
			assert.IsType(t, map[string]any{}, valueMap["a"])
			assert.Empty(t, valueMap["a"])
			assert.Len(t, valueMap, 2)
		}
	})
}

// JS: setup() > should type the snapshot state value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1550
func TestSetupTypes_Setup_StateValueOfStatechartWithNestedCompoundStates(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a"},
				{Key: "b", Initial: "b1", States: xs.States{
					{Key: "b1", Initial: "b11", States: xs.States{
						{Key: "b11"},
						{Key: "b12"},
					}},
					{Key: "b2"},
				}},
			},
		})

		snapshot := xs.CreateActor(machine).GetSnapshot()

		// Beyond JS: the value is the initial state key.
		assert.Equal(t, "a", snapshot.Value)
	})
}

// JS: setup() > state.value from setup state machine actors should be strongly-typed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1587
func TestSetupTypes_Setup_StateValueFromSetupMachineActorsIsStronglyTyped(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "green",
			States: xs.States{
				{Key: "green"},
				{Key: "yellow"},
				{Key: "red", Initial: "walk", States: xs.States{
					{Key: "walk"},
					{Key: "wait"},
					{Key: "stop"},
				}},
				{Key: "emergency", Type: xs.Parallel, States: xs.States{
					{Key: "main", Initial: "blinking", States: xs.States{
						{Key: "blinking"},
					}},
					{Key: "cross", Initial: "blinking", States: xs.States{
						{Key: "blinking"},
					}},
				}},
			},
		})

		actor := xs.CreateActor(machine).Start()

		stateValue := actor.GetSnapshot().Value

		// Beyond JS: the `satisfies` literals are type-level only; the runtime value is the initial state.
		assert.Equal(t, "green", stateValue)
	})
}

// JS: setup() > state.value is exhaustive
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1671
func TestSetupTypes_Setup_StateValueIsExhaustive(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "green",
			States: xs.States{
				{Key: "green"},
				{Key: "yellow"},
				{Key: "red", Initial: "walk", States: xs.States{
					{Key: "walk"},
					{Key: "wait"},
					{Key: "stop"},
				}},
				{Key: "emergency", Type: xs.Parallel, States: xs.States{
					{Key: "main", Initial: "blinking", States: xs.States{
						{Key: "blinking"},
					}},
					{Key: "cross", Initial: "blinking", States: xs.States{
						{Key: "blinking"},
					}},
				}},
			},
		})
		actor := xs.CreateActor(machine)
		value := actor.GetSnapshot().Value

		// Beyond JS: the narrowing branches are type-level only; at runtime
		// the first branch (`value === 'green'`) is taken.
		assert.Equal(t, "green", value)
	})
}

// JS: setup() > should accept `assign` when no actor and children types are provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1749
func TestSetupTypes_Setup_ShouldAcceptAssignWhenNoActorAndChildrenTypesAreProvided(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				// assign({}) assigns nothing: the next context is the current one.
				"RESTART": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any { return a.Context })}}},
			},
		})
	})
}

// JS: setup() > should not allow matching against any value when the machine has no states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1759
func TestSetupTypes_Setup_RejectsMatchingAnyValueWhenMachineHasNoStates(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{})

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: results per matchesState against the value {}.
		assert.True(t, snapshot.Matches(map[string]any{}))
		assert.False(t, snapshot.Matches("pending"))
		assert.False(t, snapshot.Matches(map[string]any{"foo": "pending"}))
	})
}

// JS: setup() > should allow matching against a valid string value of a simple FSM
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1780
func TestSetupTypes_Setup_ShouldAllowMatchingValidStringValueOfSimpleFSM(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2SimpleFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: the machine is in `green`.
		assert.True(t, snapshot.Matches("green"))
		assert.False(t, snapshot.Matches("yellow"))
		assert.False(t, snapshot.Matches("red"))
	})
}

// JS: setup() > should not allow matching against a invalid string value of a simple FSM
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1797
func TestSetupTypes_Setup_RejectsMatchingInvalidStringValueOfSimpleFSM(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2SimpleFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: 'orange' is not the current value.
		assert.False(t, snapshot.Matches("orange"))
	})
}

// JS: setup() > should not allow matching against an empty object value of a simple FSM
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1815
func TestSetupTypes_Setup_RejectsMatchingEmptyObjectValueOfSimpleFSM(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2SimpleFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: matchesState({}, 'green') is false (parent more specific than child).
		assert.False(t, snapshot.Matches(map[string]any{}))
	})
}

// JS: setup() > should not allow matching against an object value with a key that is a valid value of a simple FSM
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1833
func TestSetupTypes_Setup_RejectsMatchingObjectValueKeyedByValidSimpleFSMValue(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2SimpleFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: matchesState({green: {}}, 'green') is false (parent more specific than child).
		assert.False(t, snapshot.Matches(map[string]any{"green": map[string]any{}}))
	})
}

// JS: setup() > should allow matching against valid top state keys of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1853
func TestSetupTypes_Setup_ShouldAllowMatchingValidTopStateKeysNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2NestedFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: the value is {green: 'walk'}.
		assert.True(t, snapshot.Matches("green"))
		assert.False(t, snapshot.Matches("yellow"))
		assert.False(t, snapshot.Matches("red"))
	})
}

// JS: setup() > should not allow matching against an invalid top state key of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1876
func TestSetupTypes_Setup_RejectsMatchingInvalidTopStateKeyNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2NestedFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: 'orange' is not a key of the value {green: 'walk'}.
		assert.False(t, snapshot.Matches("orange"))
	})
}

// JS: setup() > should allow matching against a valid full object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1900
func TestSetupTypes_Setup_ShouldAllowMatchingValidFullObjectValueNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2NestedFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: the type accepts {green: 'wait'}; at runtime the value is {green: 'walk'}, so it does not match.
		assert.False(t, snapshot.Matches(map[string]any{"green": "wait"}))
		assert.True(t, snapshot.Matches(map[string]any{"green": "walk"}))
	})
}

// JS: setup() > should allow matching against a valid non-full object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1923
func TestSetupTypes_Setup_ShouldAllowMatchingValidNonFullObjectValueNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "green",
			States: xs.States{
				{Key: "green", Initial: "walk", States: xs.States{
					{Key: "walk", Initial: "steady", States: xs.States{
						{Key: "steady"},
						{Key: "slowingDown"},
					}},
					{Key: "wait"},
				}},
				{Key: "yellow"},
				{Key: "red"},
			},
		})

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: the value is {green: {walk: 'steady'}}; 'wait' is not a key of {walk: 'steady'}.
		assert.False(t, snapshot.Matches(map[string]any{"green": "wait"}))
		assert.True(t, snapshot.Matches(map[string]any{"green": "walk"}))
	})
}

// JS: setup() > should not allow matching against a invalid object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1952
func TestSetupTypes_Setup_RejectsMatchingInvalidObjectValueNestedCompoundL1952(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2NestedFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: matchesState('invalid', 'walk') is false.
		assert.False(t, snapshot.Matches(map[string]any{"green": "invalid"}))
	})
}

// JS: setup() > should not allow matching against a invalid object value with self-key at value position
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1976
func TestSetupTypes_Setup_RejectsMatchingInvalidObjectValueWithSelfKeyL1976(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := setupTypes2NestedFSM()

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: matchesState('green', 'walk') is false.
		assert.False(t, snapshot.Matches(map[string]any{"green": "green"}))
	})
}

// JS: setup() > should accept an after transition that references a known delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2000
func TestSetupTypes_Setup_ShouldAcceptAfterTransitionReferencingKnownDelay(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Delays: map[string]any{"hundred": ms(100)},
		}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", After: map[string]xs.Transitions{
					"hundred": {{Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should not accept an after transition that references an unknown delay when delays are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2018
func TestSetupTypes_Setup_RejectsAfterTransitionWithUnknownDelayWhenDelaysConfigured(t *testing.T) {
	// The type rejection is disabled in JS (@x-ts-expect-error, TypeScript#55709);
	// the only live check is that createMachine does not throw.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Delays: map[string]any{"thousand": ms(1000)},
		}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", After: map[string]xs.Transitions{
					"unknown": {{Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should not accept an after transition that references an unknown delay when delays are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2037
func TestSetupTypes_Setup_RejectsAfterTransitionWithUnknownDelayWhenNoDelays(t *testing.T) {
	// The type rejection is disabled in JS (@x-ts-expect-error, TypeScript#55709);
	// the only live check is that createMachine does not throw.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", After: map[string]xs.Transitions{
					"unknown": {{Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should accept a guarded transition that references a known guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2052
func TestSetupTypes_Setup_ShouldAcceptGuardedTransitionReferencingKnownGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"checkStuff": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", On: map[string]xs.Transitions{
					"NEXT": {{Guard: xs.GuardRef{Type: "checkStuff"}, Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should not accept a guarded transition that references an unknown guard when guards are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2076
func TestSetupTypes_Setup_RejectsGuardedTransitionWithUnknownGuardWhenGuardsConfigured(t *testing.T) {
	// JS rejects `guard: 'unknown'` only at type level; the runtime config is
	// accepted (guards are resolved when a transition is evaluated, not when
	// the machine is created).
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Guards: map[string]xs.Guard{
				"checkStuff": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", On: map[string]xs.Transitions{
					"NEXT": {{Guard: xs.GuardRef{Type: "unknown"}, Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should not accept a guarded transition that references an unknown guard when guards are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2101
func TestSetupTypes_Setup_RejectsGuardedTransitionWithUnknownGuardWhenNoGuards(t *testing.T) {
	// JS rejects `guard: 'checkStuff'` only at type level; the runtime config
	// is accepted (guards are resolved when a transition is evaluated).
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", On: map[string]xs.Transitions{
					"NEXT": {{Guard: xs.GuardRef{Type: "checkStuff"}, Target: "b"}},
				}},
				{Key: "b"},
			},
		})
	})
}

// JS: setup() > should accept `enqueueActions` within the config when actions are not configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2123
func TestSetupTypes_Setup_ShouldAcceptEnqueueActionsInConfigWhenActionsNotConfigured(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"SOMETHING": {{Actions: xs.Actions{
					xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
						a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
					}),
				}}},
			},
		})
	})
}

// JS: setup() > should accept `enqueueActions` within the config when empty delays are configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2145
func TestSetupTypes_Setup_ShouldAcceptEnqueueActionsInConfigWhenEmptyDelaysConfigured(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Delays: map[string]any{},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.EnqueueActions(func(xs.EnqueueArgs[any]) {})},
		})
	})
}

// JS: setup() > should accept `enqueueActions` that doesn't use any other defined actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2153
func TestSetupTypes_Setup_ShouldAcceptEnqueueActionsNotUsingOtherDefinedActions(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
				}),
			},
		})
	})
}

// JS: setup() > should accept `enqueueActions` that uses a known guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2172
func TestSetupTypes_Setup_ShouldAcceptEnqueueActionsUsingKnownGuard(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					if a.Check(xs.GuardRef{Type: "checkStuff"}) {
						a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
					}
				}),
			},
			Guards: map[string]xs.Guard{
				"checkStuff": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		})
	})
}

// JS: setup() > should not allow `enqueueActions` to use an unknown guard (when guards are configured)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2196
func TestSetupTypes_Setup_RejectsEnqueueActionsUnknownGuardWhenGuardsConfigured(t *testing.T) {
	// JS rejects check('unknown') only at type level; the enqueueActions body
	// is never executed in the test, so setup() just has to accept it.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					if a.Check(xs.GuardRef{Type: "unknown"}) {
						a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
					}
				}),
			},
			Guards: map[string]xs.Guard{
				"checkStuff": xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
			},
		})
	})
}

// JS: setup() > should not allow `enqueueActions` to use an unknown guard (when guards are not configured)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2225
func TestSetupTypes_Setup_RejectsEnqueueActionsUnknownGuardWhenNoGuards(t *testing.T) {
	// JS rejects check('unknown') only at type level; the enqueueActions body
	// is never executed in the test, so setup() just has to accept it.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
					if a.Check(xs.GuardRef{Type: "unknown"}) {
						a.Enqueue(xs.Raise(xs.Ev("SOMETHING_ELSE")))
					}
				}),
			},
		})
	})
}

// JS: setup() > should be able to use a parameterized `enqueueActions` action with its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2251
func TestSetupTypes_Setup_ShouldUseParameterizedEnqueueActionsWithRequiredParams(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(xs.EnqueueArgs[any]) {}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "doStuff", Params: 0}},
		})
	})
}

// JS: setup() > should not accept a string reference to parameterized `enqueueActions` without its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2264
func TestSetupTypes_Setup_RejectsStringRefToParameterizedEnqueueActionsWithoutParams(t *testing.T) {
	// JS rejects `entry: 'doStuff'` only at type level. Go has a single action-reference form (xs.ActionRef), so the
	// string and object forms of the JS config are the same Go value.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(xs.EnqueueArgs[any]) {}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "doStuff"}},
		})
	})
}

// JS: setup() > should not accept an object reference to parameterized `enqueueActions` without its required params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2275
func TestSetupTypes_Setup_RejectsObjectRefToParameterizedEnqueueActionsWithoutParams(t *testing.T) {
	// JS rejects `entry: {type: 'doStuff'}` only at type level. Go has a single action-reference form (xs.ActionRef), so the
	// string and object forms of the JS config are the same Go value.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(xs.EnqueueArgs[any]) {}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "doStuff"}},
		})
	})
}

// JS: setup() > should not accept an object reference to parameterized `enqueueActions` without its required params in the machine
// (second JS test with the identical name, JS L2288)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2288
func TestSetupTypes_Setup_RejectsObjectRefToParameterizedEnqueueActionsWithoutParams2(t *testing.T) {
	// JS rejects `entry: {type: 'doStuff'}` only at type level (duplicate of the previous JS test). Go has a single action-reference form (xs.ActionRef), so the
	// string and object forms of the JS config are the same Go value.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(xs.EnqueueArgs[any]) {}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "doStuff"}},
		})
	})
}

// JS: setup() > should not accept a reference to parameterized `enqueueActions` with wrong params in the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2301
func TestSetupTypes_Setup_RejectsRefToParameterizedEnqueueActionsWithWrongParams(t *testing.T) {
	// JS rejects `params: 'foo'` only at type level; Go Params is `any`. Go has a single action-reference form (xs.ActionRef), so the
	// string and object forms of the JS config are the same Go value.
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{
				"doStuff": xs.EnqueueActions(func(xs.EnqueueArgs[any]) {}),
			},
		}).CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "doStuff", Params: "foo"}},
		})
	})
}

// JS: setup() > should allow `log` action to be configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2315
func TestSetupTypes_Setup_ShouldAllowLogActionToBeConfigured(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{"writeDown": xs.Log("foo")},
		})
	})
}

// JS: setup() > should allow `cancel` action to be configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2323
func TestSetupTypes_Setup_ShouldAllowCancelActionToBeConfigured(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{"revert": xs.Cancel("foo")},
		})
	})
}

// JS: setup() > should allow `stopChild` action to be configured
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2331
func TestSetupTypes_Setup_ShouldAllowStopChildActionToBeConfigured(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.NewSetup[any](xs.Implementations{
			Actions: map[string]xs.Action{"releaseFromDuty": xs.StopChild("foo")},
		})
	})
}

// JS: setup() > EventFrom should work with a machine that has transitions defined on a state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2339
func TestSetupTypes_Setup_EventFromWorksWithMachineWithTransitionsOnAState(t *testing.T) {
	type ctx struct{ MyVar string }

	// The trailing `(_accept: EventFrom<typeof machine>) => {}` call is a
	// type-level check with no Go equivalent; the machine creation runs.
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
			ID:      "authorization",
			Initial: "loading",
			Context: ctx{MyVar: "foo"},
			States: xs.States{
				{Key: "loaded"},
				{Key: "loading", On: map[string]xs.Transitions{
					"SOME_EVENT": {{Target: "loaded"}},
				}},
			},
		})
	})
}

// JS: setup() > ContextFrom should work with a machine that has transitions defined on a state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2369
func TestSetupTypes_Setup_ContextFromWorksWithMachineWithTransitionsOnAState(t *testing.T) {
	type ctx struct{ MyVar string }

	// The trailing `(_accept: ContextFrom<typeof machine>) => {}` call is a
	// type-level check; Go's context type is the type parameter C. The machine
	// creation runs.
	assert.NotPanics(t, func() {
		xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
			ID:      "authorization",
			Initial: "loading",
			Context: ctx{MyVar: "foo"},
			States: xs.States{
				{Key: "loaded"},
				{Key: "loading", On: map[string]xs.Transitions{
					"SOME_EVENT": {{Target: "loaded"}},
				}},
			},
		})
	})
}

// JS: setup() > should strongly type the state IDs in snapshot.getMeta()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2399
func TestSetupTypes_Setup_ShouldStronglyTypeStateIDsInGetMeta(t *testing.T) {
	assert.NotPanics(t, func() {
		machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
			ID:      "root",
			Initial: "parentState",
			States: xs.States{
				{
					Key:     "parentState",
					Meta:    map[string]any{},
					Initial: "childState",
					States: xs.States{
						{Key: "childState", Meta: map[string]any{}},
						{Key: "stateWithId", ID: "state with id", Meta: map[string]any{}},
					},
				},
			},
		})

		actor := xs.CreateActor(machine)

		metaValues := actor.GetSnapshot().GetMeta()

		// Beyond JS: JS only reads these keys (the typed-key rejections are
		// type-level). machineSnapshotGetMeta lists active nodes that have
		// meta: parentState and childState. `root` has no meta and
		// `state with id` is not active.
		assert.Contains(t, metaValues, "root.parentState")
		assert.Contains(t, metaValues, "root.parentState.childState")
		assert.NotContains(t, metaValues, "root")
		assert.NotContains(t, metaValues, "state with id")
	})
}

// JS: createStateConfig > should be able to create a state config with a custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2438
func TestSetupTypes_CreateStateConfig_ShouldCreateStateConfigWithCustomAction(t *testing.T) {
	type ctx struct{ Count int }

	assert.NotPanics(t, func() {
		machineSetup := xs.NewSetup[ctx](xs.Implementations{
			Actions: map[string]xs.Action{
				"doSomething": xs.ActionFunc(func(xs.ActionArgs[ctx]) {}),
			},
			Guards: map[string]xs.Guard{
				"isLightActive": xs.GuardFunc(func(xs.GuardArgs[ctx]) bool { return true }),
			},
		})

		timerOn := func() map[string]xs.Transitions {
			return map[string]xs.Transitions{
				"timer": {{
					Actions: xs.Actions{xs.ActionRef{Type: "doSomething"}},
					Guard:   xs.GuardRef{Type: "isLightActive"},
				}},
			}
		}

		green := machineSetup.CreateStateConfig(xs.StateConfig{Key: "green", On: timerOn()})
		yellow := machineSetup.CreateStateConfig(xs.StateConfig{Key: "yellow", On: timerOn()})
		red := machineSetup.CreateStateConfig(xs.StateConfig{Key: "red", On: timerOn()})

		// JS marks the next three with @ts-expect-error (type-only rejection);
		// they are still valid runtime configs and are part of the machine.
		invalidEvent := machineSetup.CreateStateConfig(xs.StateConfig{
			Key: "invalidEvent",
			On:  map[string]xs.Transitions{"nonsense": {{}}},
		})
		invalidAction := machineSetup.CreateStateConfig(xs.StateConfig{
			Key: "invalidAction",
			On: map[string]xs.Transitions{
				"timer": {{Actions: xs.Actions{xs.ActionRef{Type: "nonexistent"}}}},
			},
		})
		invalidGuard := machineSetup.CreateStateConfig(xs.StateConfig{
			Key: "invalidGuard",
			On: map[string]xs.Transitions{
				"timer": {{Guard: xs.GuardRef{Type: "nonexistent"}}},
			},
		})

		machineSetup.CreateMachine(xs.MachineConfig[ctx]{
			Context: ctx{Count: 0},
			Initial: "green",
			States: xs.States{
				green,
				yellow,
				red,
				invalidEvent,
				invalidAction,
				invalidGuard,
			},
		})
	})
}

// JS: createStateConfig > should allow matching against valid top state keys of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2527
func TestSetupTypes_CreateStateConfig_ShouldAllowMatchingValidTopStateKeysNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machineSetup := xs.NewSetup[any](xs.Implementations{})
		green := machineSetup.CreateStateConfig(xs.StateConfig{
			Key:     "green",
			Initial: "walk",
			States: xs.States{
				{Key: "walk"},
				{Key: "wait"},
			},
		})
		machine := machineSetup.CreateMachine(xs.MachineConfig[any]{
			Initial: "green",
			States: xs.States{
				green,
				{Key: "yellow"},
				{Key: "red"},
			},
		})

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: the value is {green: 'walk'}.
		assert.True(t, snapshot.Matches("green"))
		assert.False(t, snapshot.Matches("yellow"))
		assert.False(t, snapshot.Matches("red"))
	})
}

// JS: createStateConfig > should not allow matching against an invalid top state key of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2552
func TestSetupTypes_CreateStateConfig_RejectsMatchingInvalidTopStateKeyNestedCompound(t *testing.T) {
	assert.NotPanics(t, func() {
		machineSetup := xs.NewSetup[any](xs.Implementations{})
		green := machineSetup.CreateStateConfig(xs.StateConfig{
			Key:     "green",
			Initial: "walk",
			States: xs.States{
				{Key: "walk"},
				{Key: "wait"},
			},
		})
		machine := machineSetup.CreateMachine(xs.MachineConfig[any]{
			Initial: "green",
			States: xs.States{
				green,
				{Key: "yellow"},
				{Key: "red"},
			},
		})

		snapshot := xs.CreateActor(machine).Start().GetSnapshot()

		// Beyond JS: 'orange' is not a key of the value {green: 'walk'}.
		assert.False(t, snapshot.Matches("orange"))
	})
}

// Every test in setup.types.test.ts L2578-3197 is a TypeScript type-level
// test: none calls expect(); each one only checks that a snippet type-checks
// or that a `// @ts-expect-error` line is rejected by the compiler.

// JS: setup() > should allow matching against a valid full object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1900
func TestSetupTypes_Setup_MatchesValidFullObjectValueNestedCompound(t *testing.T) {
	t.Skip("N/A: type-level only — createStateConfig state + snapshot.matches({green: 'wait'}) type-checks; no runtime expectations")
}

// JS: setup() > should allow matching against a valid non-full object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1923
func TestSetupTypes_Setup_MatchesValidNonFullObjectValueNestedCompound(t *testing.T) {
	t.Skip("N/A: type-level only — snapshot.matches({green: 'wait'}) accepted for a non-full value with deeper nesting; no runtime expectations")
}

// JS: setup() > should not allow matching against a invalid object value of a statechart with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1952
func TestSetupTypes_Setup_RejectsMatchingInvalidObjectValueNestedCompound(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on snapshot.matches({green: 'invalid'}); no runtime expectations")
}

// JS: setup() > should not allow matching against a invalid object value with self-key at value position
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L1976
func TestSetupTypes_Setup_RejectsMatchingInvalidObjectValueWithSelfKeyAtValue(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on snapshot.matches({green: 'green'}); no runtime expectations")
}

// JS: extend > undefined actions handling > should error on undefined actions in createMachine without extend
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2689
func TestSetupTypes_Extend_UndefinedActions_ErrorsInCreateMachineWithoutExtend(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on setup({}).createMachine({entry: 'nonexistent'}); no runtime expectations")
}

// JS: extend > undefined actions handling > should error on undefined actions in createMachine with empty extend
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2696
func TestSetupTypes_Extend_UndefinedActions_ErrorsInCreateMachineWithEmptyExtend(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on setup({}).extend({}).createMachine({entry: 'nonexistent'}); no runtime expectations")
}

// JS: extend > undefined actions handling > should error on undefined actions in extend enqueueActions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2703
func TestSetupTypes_Extend_UndefinedActions_ErrorsInExtendEnqueueActions(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error on enqueue('nonexistent') inside extend actions; no runtime expectations")
}

// JS: extend > actions > should allow extending actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2716
func TestSetupTypes_Extend_Actions_ShouldAllowExtendingActions(t *testing.T) {
	t.Skip("N/A: type-level only — action 'foo' added via extend() is accepted as entry: 'foo'; no runtime expectations")
}

// JS: extend > actions > should allow referencing base actions in extended actions via enqueueActions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2728
func TestSetupTypes_Extend_Actions_ReferencingBaseActionsViaEnqueueActions(t *testing.T) {
	t.Skip("N/A: type-level only — enqueue.raise inside an extended enqueueActions action type-checks; no runtime expectations")
}

// JS: extend > guards > should allow extending guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2749
func TestSetupTypes_Extend_Guards_ShouldAllowExtendingGuards(t *testing.T) {
	t.Skip("N/A: type-level only — extended guard 'truthy' accepted, @ts-expect-error on unknown guard 'notTruthy'; no runtime expectations")
}

// JS: extend > guards > should allow referencing base guards in extended guards with not
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2769
func TestSetupTypes_Extend_Guards_ReferencingBaseGuardsWithNot(t *testing.T) {
	t.Skip("N/A: type-level only — not('truthy') in extend accepted, @ts-expect-error on not('existent') and guard 'notNotNotTruthy'; no runtime expectations")
}

// JS: extend > guards > should allow referencing extended guards in further extended guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2795
func TestSetupTypes_Extend_Guards_ReferencingExtendedGuardsInFurtherExtend(t *testing.T) {
	t.Skip("N/A: type-level only — and/or over base+extended guard names accepted, @ts-expect-error on 'existent' and 'fake'; no runtime expectations")
}

// JS: extend > guards > should allow referencing extended guards in extended actions via check
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2830
func TestSetupTypes_Extend_Guards_ReferencingExtendedGuardsViaCheck(t *testing.T) {
	t.Skip("N/A: type-level only — check('truthy'/'alsoTruthy') accepted, @ts-expect-error on check('nonexistent'); no runtime expectations")
}

// JS: extend > delays > should allow extending delays
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2856
func TestSetupTypes_Extend_Delays_ShouldAllowExtendingDelays(t *testing.T) {
	t.Skip("N/A: type-level only — delay 'medium' added via extend() accepted in after; no runtime expectations")
}

// JS: extend > delays > should allow referencing base delays in extended delays
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2876
func TestSetupTypes_Extend_Delays_ReferencingBaseDelaysInExtendedDelays(t *testing.T) {
	t.Skip("N/A: type-level only — raise delays 'short'/'medium' accepted, @ts-expect-error on delay 'nonexistent'; no runtime expectations")
}

// JS: extend > delays > should allow referencing extended delays in further extended delays
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2911
func TestSetupTypes_Extend_Delays_ReferencingExtendedDelaysInFurtherExtend(t *testing.T) {
	t.Skip("N/A: type-level only — delays from setup + two extend() calls accepted, @ts-expect-error on after key 'nonexistent'; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe action action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2960
func TestSetupTypes_TypeBoundActions_TypeSafeAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.createAction args typed by setup types; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe assign action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L2997
func TestSetupTypes_TypeBoundActions_TypeSafeAssignAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.assign context typed by setup types; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe raise action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3028
func TestSetupTypes_TypeBoundActions_TypeSafeRaiseAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.raise event typed by setup types; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe sendTo action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3060
func TestSetupTypes_TypeBoundActions_TypeSafeSendToAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.sendTo event typed by setup types; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe log action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3090
func TestSetupTypes_TypeBoundActions_TypeSafeLogAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.log context/event typed by setup types (satisfies checks); no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe cancel action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3109
func TestSetupTypes_TypeBoundActions_TypeSafeCancelAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.cancel('some-id') usable as entry; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe stopChild action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3115
func TestSetupTypes_TypeBoundActions_TypeSafeStopChildAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.stopChild('child') usable as entry; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe enqueueActions action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3121
func TestSetupTypes_TypeBoundActions_TypeSafeEnqueueActionsAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.enqueueActions check/enqueue typed by setup; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe emit action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3161
func TestSetupTypes_TypeBoundActions_TypeSafeEmitAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.emit typed by setup emitted types; @ts-expect-error on use in another setup; no runtime expectations")
}

// JS: type-bound actions > should be able to create a type-safe spawnChild action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/setup.types.test.ts#L3179
func TestSetupTypes_TypeBoundActions_TypeSafeSpawnChildAction(t *testing.T) {
	t.Skip("N/A: type-level only — machineSetup.spawnChild('child') typed by setup actors; @ts-expect-error on use in another setup; no runtime expectations")
}
