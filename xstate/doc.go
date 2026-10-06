// Package xstate implements state machines and actors using XState semantics.
//
// Define a machine with CreateMachine and MachineConfig, then run it with
// CreateActor. Actors accept events through Send and expose typed snapshots
// through GetSnapshot. States is an ordered slice so state entry and exit follow
// declaration order. Assign returns the next context value; callers should copy
// mutable maps and slices before changing them.
//
// FromPromise, FromCallback, FromObservable, and FromTransition create other
// actor kinds. Promise bodies run on goroutines and receive a context cancelled
// when the actor stops. Actions and observers run synchronously under the actor
// system's lock; they should not wait for another goroutine to call that system.
//
// SendTo between built-in actors in separate root systems delivers after the
// calling goroutine releases its outermost system lock. Delivery and synchronous
// replies finish before the outermost actor call returns. An action or observer
// reading the foreign snapshot before that unlock does not yet see the send;
// source snapshot notifications precede foreign delivery. Concurrent calls may
// interleave after unlocking. Same-system sends retain their reentrant mailbox
// behavior. Calling a foreign actor's Send or GetSnapshot directly from an
// action still acquires a nested lock; reciprocal direct calls can deadlock.
//
// The import path is github.com/nguyenvanduocit/go-xstate/xstate. Companion
// packages store, graph, and scxml are separate packages in the same module.
package xstate
