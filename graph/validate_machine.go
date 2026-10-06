package graph

import (
	"errors"
	"reflect"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func validateState(state *xs.StateNode) {
	if len(state.Invoke()) > 0 {
		panic(errors.New("Invocations on test machines are not supported"))
	}
	if len(state.After()) > 0 {
		panic(errors.New("After events on test machines are not supported"))
	}
	// Like JS, this doesn't account for always transitions, and only checks
	// inline actions, not referenced ones.
	actions := append(append(xs.Actions{}, state.Entry...), state.Exit...)
	for _, t := range state.TransitionList() {
		actions = append(actions, t.Actions...)
	}
	for _, action := range actions {
		if delay, ok := xs.ActionDelay(action); ok && isNumber(delay) {
			panic(errors.New("Delayed actions on test machines are not supported"))
		}
	}

	for _, child := range state.ChildStates() {
		validateState(child)
	}
}

// isNumber mirrors `typeof delay === 'number'` (a time.Duration or any other
// numeric value; delay names and delay expressions are not numbers).
func isNumber(v any) bool {
	switch reflect.ValueOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// ValidateMachine mirrors validateMachine(machine). It panics with
// errors.New(msg) for invocations ("Invocations on test machines are not
// supported"), delayed transitions ("After events on test machines are not
// supported") and delayed actions ("Delayed actions on test machines are not
// supported").
func ValidateMachine(machine xs.AnyStateMachine) {
	validateState(machine.RootNode())
}
