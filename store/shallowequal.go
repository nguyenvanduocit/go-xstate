package store

import (
	"math"
	"reflect"
)

// ShallowEqual mirrors shallowEqual(a, b): Object.is on the values, else
// equal key sets (maps, struct fields) whose values are Object.is-equal.
// Slices compare like JS arrays (same length, Object.is-equal elements);
// pointers to structs, maps or slices compare their targets shallowly.
func ShallowEqual(a, b any) bool {
	if objectIs(a, b) {
		return true
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() || va.Type() != vb.Type() {
		return false
	}
	if va.Kind() == reflect.Pointer {
		if va.IsNil() || vb.IsNil() {
			return false
		}
		va, vb = va.Elem(), vb.Elem()
	}
	switch va.Kind() {
	case reflect.Struct:
		for i := 0; i < va.NumField(); i++ {
			if !sameValue(va.Field(i), vb.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if va.IsNil() || vb.IsNil() || va.Len() != vb.Len() {
			return false
		}
		iter := va.MapRange()
		for iter.Next() {
			other := vb.MapIndex(iter.Key())
			if !other.IsValid() || !sameValue(iter.Value(), other) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if va.Kind() == reflect.Slice && (va.IsNil() || vb.IsNil()) {
			return false
		}
		if va.Len() != vb.Len() {
			return false
		}
		for i := 0; i < va.Len(); i++ {
			if !sameValue(va.Index(i), vb.Index(i)) {
				return false
			}
		}
		return true
	}
	return false
}

// objectIs mirrors JS Object.is for Go values: scalars and strings compare by
// value (NaN equals NaN), maps, slices, channels and pointers compare
// by identity, and inline aggregates (structs, arrays) compare field by
// field with the same rule. A Go struct value has no reference identity, so
// two structs holding identical fields are "the same object". Non-nil
// functions compare unequal because Go does not expose closure identity.
func objectIs(a, b any) bool {
	return sameValue(reflect.ValueOf(a), reflect.ValueOf(b))
}

func sameValue(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return sameValue(a.Elem(), b.Elem())
	case reflect.Func:
		return a.IsNil() && b.IsNil()
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.UnsafePointer:
		return a.Pointer() == b.Pointer()
	case reflect.Slice:
		return a.Pointer() == b.Pointer() && a.Len() == b.Len() && a.Cap() == b.Cap()
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !sameValue(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			if !sameValue(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Float32, reflect.Float64:
		x, y := a.Float(), b.Float()
		if math.IsNaN(x) && math.IsNaN(y) {
			return true
		}
		if x == 0 && y == 0 {
			return math.Signbit(x) == math.Signbit(y)
		}
		return x == y
	case reflect.Complex64, reflect.Complex128:
		return a.Complex() == b.Complex()
	case reflect.String:
		return a.String() == b.String()
	case reflect.Bool:
		return a.Bool() == b.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() == b.Uint()
	}
	return false
}
