package store_test

// Shared test helpers used by every translated store test file. Translators
// must use these instead of defining their own equivalents.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
)

// ---- vi.fn() ----

// spy mirrors vi.fn(): it records every call's arguments.
type spy struct {
	mu    sync.Mutex
	calls [][]any
}

func newSpy() *spy { return &spy{} }

// Call records a call.
func (s *spy) Call(args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, args)
}

// Calls returns a copy of recorded calls.
func (s *spy) Calls() [][]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]any, len(s.calls))
	copy(out, s.calls)
	return out
}

// Count returns the number of calls (toHaveBeenCalledTimes).
func (s *spy) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// Reset mirrors spy.mockClear().
func (s *spy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = nil
}

// ---- Promise.withResolvers / await ----

// signal is a one-shot completion used where JS tests await
// Promise.withResolvers().promise.
type signal struct {
	once sync.Once
	ch   chan struct{}
}

func newSignal() *signal { return &signal{ch: make(chan struct{})} }

func (s *signal) Resolve() { s.once.Do(func() { close(s.ch) }) }

// Wait fails the test if the signal is not resolved within d (default 2s).
func (s *signal) Wait(t testing.TB, d ...time.Duration) {
	t.Helper()
	timeout := 2 * time.Second
	if len(d) > 0 {
		timeout = d[0]
	}
	select {
	case <-s.ch:
	case <-time.After(timeout):
		t.Fatalf("timed out after %v waiting for signal", timeout)
	}
}

// sleep mirrors `await sleep(ms)` / `await new Promise(r => setTimeout(r, ms))`.
func sleep(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// ms converts JS milliseconds to time.Duration.
func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// panicMessage runs fn and returns the recovered panic message ("" if fn
// did not panic). error values yield err.Error().
func panicMessage(fn func()) (msg string) {
	defer func() {
		r := recover()
		switch v := r.(type) {
		case nil:
			msg = ""
		case error:
			msg = v.Error()
		default:
			msg = fmt.Sprint(v)
		}
	}()
	fn()
	return ""
}

// recovered runs fn and returns the recovered panic value (nil if none);
// mirrors the `getThrown(fn)` helper of validate.test.ts.
func recovered(fn func()) (r any) {
	defer func() { r = recover() }()
	fn()
	return nil
}

// strPtr returns &s (storage values; nil mirrors null).
func strPtr(s string) *string { return &s }

// ---- storage ----

// memoryStorage mirrors createMockStorage() / createMemoryStorage() /
// createStorage(): synchronous in-memory StateStorage.
type memoryStorage struct {
	mu   sync.Mutex
	data map[string]string
}

func newMemoryStorage() *memoryStorage { return &memoryStorage{data: map[string]string{}} }

func (m *memoryStorage) GetItem(name string) (*string, *xstore.Promise[*string]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.data[name]; ok {
		return &v, nil
	}
	return nil, nil
}

func (m *memoryStorage) SetItem(name, value string) *xstore.Promise[struct{}] {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[name] = value
	return nil
}

func (m *memoryStorage) RemoveItem(name string) *xstore.Promise[struct{}] {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, name)
	return nil
}

// asyncMemoryStorage mirrors createAsyncMockStorage(): every method returns
// an already-settled promise, so persist treats it as async storage.
type asyncMemoryStorage struct{ mem *memoryStorage }

func newAsyncMemoryStorage() *asyncMemoryStorage {
	return &asyncMemoryStorage{mem: newMemoryStorage()}
}

func (a *asyncMemoryStorage) GetItem(name string) (*string, *xstore.Promise[*string]) {
	v, _ := a.mem.GetItem(name)
	p, resolve, _ := xstore.NewPromise[*string]()
	resolve(v)
	return nil, p
}

func (a *asyncMemoryStorage) SetItem(name, value string) *xstore.Promise[struct{}] {
	a.mem.SetItem(name, value)
	p, resolve, _ := xstore.NewPromise[struct{}]()
	resolve(struct{}{})
	return p
}

func (a *asyncMemoryStorage) RemoveItem(name string) *xstore.Promise[struct{}] {
	a.mem.RemoveItem(name)
	p, resolve, _ := xstore.NewPromise[struct{}]()
	resolve(struct{}{})
	return p
}

// getItem reads a storage value, awaiting async storage (JS
// `await storage.getItem(name)`). nil mirrors null.
func getItem(t testing.TB, s xstore.Storage, name string) *string {
	t.Helper()
	v, p := s.GetItem(name)
	if p == nil {
		return v
	}
	v, err := p.Wait()
	if err != nil {
		t.Fatalf("getItem(%q) rejected: %v", name, err)
	}
	return v
}

// storedJSON mirrors `JSON.parse(storage.getItem(name))`; nil when absent.
// Numbers decode as float64.
func storedJSON(t testing.TB, s xstore.Storage, name string) any {
	t.Helper()
	v := getItem(t, s, name)
	if v == nil {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(*v), &out); err != nil {
		t.Fatalf("storedJSON(%q): %v", name, err)
	}
	return out
}

// toJSON mirrors JSON.stringify(v).
func toJSON(t testing.TB, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("toJSON: %v", err)
	}
	return string(b)
}

// ---- zod stand-ins (Standard Schema) ----
//
// Values are normalized through encoding/json before checking, so struct
// contexts validate by their JSON field names (use json tags) and every
// number kind is a "number".

type zCheck func(v any, path []any) []xstore.SchemaIssue

type zSchema struct {
	check    zCheck
	optional bool
	async    bool
}

func (z *zSchema) Validate(value any) (xstore.SchemaResult, *xstore.Promise[xstore.SchemaResult]) {
	res := xstore.SchemaResult{Value: value, Issues: z.check(zNormalize(value), nil)}
	if !z.async {
		return res, nil
	}
	p, resolve, _ := xstore.NewPromise[xstore.SchemaResult]()
	resolve(res)
	return xstore.SchemaResult{}, p
}

func zNormalize(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

func zIssue(path []any, format string, args ...any) []xstore.SchemaIssue {
	return []xstore.SchemaIssue{{Message: fmt.Sprintf(format, args...), Path: path}}
}

// zNumber mirrors z.number().
func zNumber() *zSchema {
	return &zSchema{check: func(v any, path []any) []xstore.SchemaIssue {
		if _, ok := v.(float64); !ok {
			return zIssue(path, "expected number, received %T", v)
		}
		return nil
	}}
}

// zNumberMax mirrors z.number().max(n).
func zNumberMax(n float64) *zSchema {
	return &zSchema{check: func(v any, path []any) []xstore.SchemaIssue {
		f, ok := v.(float64)
		if !ok {
			return zIssue(path, "expected number, received %T", v)
		}
		if f > n {
			return zIssue(path, "too big: expected number to be <=%v", n)
		}
		return nil
	}}
}

// zString mirrors z.string().
func zString() *zSchema {
	return &zSchema{check: func(v any, path []any) []xstore.SchemaIssue {
		if _, ok := v.(string); !ok {
			return zIssue(path, "expected string, received %T", v)
		}
		return nil
	}}
}

// zLiteral mirrors z.literal(v).
func zLiteral(lit any) *zSchema {
	want := zNormalize(lit)
	return &zSchema{check: func(v any, path []any) []xstore.SchemaIssue {
		if !reflect.DeepEqual(v, want) {
			return zIssue(path, "expected %v", lit)
		}
		return nil
	}}
}

// zOptional mirrors schema.optional(): the key may be absent or null.
func zOptional(s *zSchema) *zSchema {
	return &zSchema{optional: true, check: func(v any, path []any) []xstore.SchemaIssue {
		if v == nil {
			return nil
		}
		return s.check(v, path)
	}}
}

// zObject mirrors z.object(shape): unknown keys are allowed (zod strips them).
func zObject(shape map[string]*zSchema) *zSchema {
	return &zSchema{check: func(v any, path []any) []xstore.SchemaIssue {
		m, ok := v.(map[string]any)
		if !ok {
			return zIssue(path, "expected object, received %T", v)
		}
		var issues []xstore.SchemaIssue
		for key, field := range shape {
			fieldPath := append(append([]any{}, path...), key)
			fv, present := m[key]
			if !present && !field.optional {
				issues = append(issues, zIssue(fieldPath, "required")...)
				continue
			}
			issues = append(issues, field.check(fv, fieldPath)...)
		}
		return issues
	}}
}

// zAsync mirrors schema.refine(async () => true): validation returns a
// promise (unsupported by validateSchemas).
func zAsync(s *zSchema) *zSchema {
	return &zSchema{check: s.check, optional: s.optional, async: true}
}
