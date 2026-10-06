package store

import (
	"sync/atomic"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// FromStore mirrors fromStore(config): store logic usable as xstate actor
// logic (xs.CreateActor(FromStore(cfg), xs.WithInput(42))). The actor's
// input is passed to config.ContextFn; emitted events reach actor.On.
func FromStore[C any](config StoreConfig[C]) *xs.Logic[*StoreSnapshot[C]] {
	transition := CreateStoreTransition(config.On)

	return &xs.Logic[*StoreSnapshot[C]]{
		Transition: func(snapshot *StoreSnapshot[C], event xs.Event, scope *xs.ActorScope) *StoreSnapshot[C] {
			result := transition(snapshot, event)
			next := result.Snapshot

			// The actor only commits `next` after this function returns, so
			// synchronous effects must read it directly; effects that run
			// later read the latest committed snapshot from the actor. This
			// matches CreateStore, where the snapshot is committed before
			// effects run.
			var committed atomic.Bool
			effectEnqueue := &StoreEffectEnqueue[C]{
				Send: func(ev xs.Event) { scope.Self.Send(ev) },
				Trigger: func(eventType string, payload ...xs.E) {
					scope.Self.Send(toEvent(eventType, payload))
				},
				GetSnapshot: func() *StoreSnapshot[C] {
					if committed.Load() {
						return scope.Self.AnySnapshot().(*StoreSnapshot[C])
					}
					return next
				},
			}

			for _, effect := range result.Effects {
				if effect.Run != nil {
					effect.Run(effectEnqueue)
				} else if effect.Emitted != nil {
					scope.Emit(effect.Emitted)
				}
			}

			committed.Store(true)
			return next
		},
		GetInitialSnapshot: func(_ *xs.ActorScope, input any) *StoreSnapshot[C] {
			ctx := config.Context
			if config.ContextFn != nil {
				ctx = config.ContextFn(input)
			}
			return &StoreSnapshot[C]{Status: xs.StatusActive, Context: ctx}
		},
		GetPersistedSnapshot: func(s *StoreSnapshot[C]) any { return s },
	}
}
