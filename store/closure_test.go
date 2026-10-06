package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
)

//go:noinline
func capturedValue(n int) func() int { return func() int { return n } }

func TestAtomReplacesClosuresWithDifferentCaptures(t *testing.T) {
	atom := xstore.CreateAtom(capturedValue(1))
	var observed []int
	sub := atom.SubscribeNext(func(value func() int) { observed = append(observed, value()) })
	defer sub.Unsubscribe()
	atom.Set(capturedValue(2))
	if got := atom.Get()(); got != 2 {
		t.Fatalf("updated closure returned %d, want 2", got)
	}
	if len(observed) != 1 || observed[0] != 2 {
		t.Fatalf("observed %v, want [2]", observed)
	}
}

func TestShallowEqualFunctions(t *testing.T) {
	var nilFunction func() int
	first, second := capturedValue(1), capturedValue(2)
	for _, test := range []struct {
		name string
		a, b any
		want bool
	}{
		{"nil functions", nilFunction, nilFunction, true},
		{"nil and nonnil", nilFunction, first, false},
		{"distinct captures", first, second, false},
		{"same nonnil function", first, first, false},
		{"struct fields", struct{ Fn func() int }{first}, struct{ Fn func() int }{second}, false},
		{"map values", map[string]any{"fn": first}, map[string]any{"fn": second}, false},
		{"slice elements", []any{first}, []any{second}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := xstore.ShallowEqual(test.a, test.b); got != test.want {
				t.Fatalf("ShallowEqual = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAtomFunctionComparator(t *testing.T) {
	atom := xstore.CreateAtom(capturedValue(1), xstore.AtomOptions[func() int]{
		Compare: func(a, b func() int) bool { return a() == b() },
	})
	notifications := 0
	sub := atom.SubscribeNext(func(func() int) { notifications++ })
	defer sub.Unsubscribe()
	atom.Set(capturedValue(1))
	atom.Set(capturedValue(2))
	if notifications != 1 || atom.Get()() != 2 {
		t.Fatalf("notifications = %d, value = %d; want 1 and 2", notifications, atom.Get()())
	}
}
