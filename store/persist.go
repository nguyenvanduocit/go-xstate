package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Storage mirrors StateStorage. Synchronous adapters return a nil promise;
// an adapter returning a non-nil promise is treated as async storage (JS
// `result instanceof Promise`). A nil value from GetItem mirrors null. A
// panic mirrors a throw.
type Storage interface {
	GetItem(name string) (*string, *Promise[*string])
	SetItem(name, value string) *Promise[struct{}]
	RemoveItem(name string) *Promise[struct{}]
}

// StorageFuncs adapts functions to Storage (JS object-literal storage). A nil
// func is a synchronous no-op (GetItem: null).
type StorageFuncs struct {
	GetItemFunc    func(name string) (*string, *Promise[*string])
	SetItemFunc    func(name, value string) *Promise[struct{}]
	RemoveItemFunc func(name string) *Promise[struct{}]
}

func (s StorageFuncs) GetItem(name string) (*string, *Promise[*string]) {
	if s.GetItemFunc == nil {
		return nil, nil
	}
	return s.GetItemFunc(name)
}

func (s StorageFuncs) SetItem(name, value string) *Promise[struct{}] {
	if s.SetItemFunc == nil {
		return nil
	}
	return s.SetItemFunc(name, value)
}

func (s StorageFuncs) RemoveItem(name string) *Promise[struct{}] {
	if s.RemoveItemFunc == nil {
		return nil
	}
	return s.RemoveItemFunc(name)
}

// PersistStorageValue mirrors PersistStorageValue (snapshot strategy).
type PersistStorageValue struct {
	Context any `json:"context"`
	Version any `json:"version"` // string or number
}

// PersistEventStorageValue mirrors PersistEventStorageValue (event strategy).
// Events read back from JSON are xs.E values whose integral numbers are int
// (other numbers float64), so handlers read replayed payloads like live ones.
// Checkpoint is always written (null when there is none), as the JS
// implementation does.
type PersistEventStorageValue struct {
	Events     []xs.Event `json:"events"`
	Version    any        `json:"version"`
	Checkpoint any        `json:"checkpoint"`
}

// PersistStrategy mirrors the persist `strategy` option.
type PersistStrategy string

const (
	PersistSnapshot PersistStrategy = "snapshot" // default ("" behaves as "snapshot")
	PersistEvent    PersistStrategy = "event"
)

// PersistOptions mirrors PersistOptions (PersistSnapshotOptions |
// PersistEventOptions in one struct). Fields marked "snapshot only" or
// "event only" are ignored by the other strategy. The default serializer is
// encoding/json, so C must round-trip through JSON (use json tags).
type PersistOptions[C any] struct {
	Name string
	// Storage mirrors `storage`; nil mirrors the default localStorage when it
	// is unavailable: a no-op storage.
	Storage Storage
	// Version mirrors `version` (string or number); nil means 0.
	Version any
	// Throttle mirrors `throttle` (minimum time between writes).
	Throttle time.Duration
	// Clock schedules throttled writes; nil uses real time. Pass an
	// xs.SimulatedClock where JS uses vi.useFakeTimers().
	Clock         xs.Clock
	OnDone        func(data any)
	OnError       func(err any)
	SkipHydration bool
	Strategy      PersistStrategy

	// Filter mirrors `filter` (snapshot only).
	Filter func(event xs.Event) bool
	// Pick mirrors `pick` (snapshot only); returns the JSON-able subset.
	Pick func(ctx C) any
	// Migrate mirrors `migrate` (snapshot only).
	Migrate func(persisted any, version any) C
	// Merge mirrors `merge` (snapshot only); nil shallow-merges the persisted
	// JSON object onto the current context.
	Merge       func(persisted any, current C) C
	Serialize   func(value PersistStorageValue) string // snapshot only
	Deserialize func(s string) PersistStorageValue     // snapshot only

	// MaxEvents mirrors `maxEvents` (event only); 0 means Infinity.
	MaxEvents int
	// MigrateEvents mirrors `migrate` (event only).
	MigrateEvents     func(events []xs.Event, version any) []xs.Event
	SerializeEvents   func(value PersistEventStorageValue) string // event only
	DeserializeEvents func(s string) PersistEventStorageValue     // event only
}

// Extension-state keys of persist. JS uses the string keys `_persist`,
// `_persistEvents`, `_persistCheckpoint` and three symbols.
const (
	persistKeyInternals  = "@@xstate-store-persist"
	persistKeyRevision   = "@@xstate-store-persist-revision"
	persistKeyClearEpoch = "@@xstate-store-persist-clear-epoch"
	persistKeyStatus     = "_persist"
	persistKeyEvents     = "_persistEvents"
	persistKeyCheckpoint = "_persistCheckpoint"
)

const persistRehydrateEventType = "__persist.rehydrate"

// persistStatus mirrors `_persist: { hydrated }`.
type persistStatus struct{ Hydrated bool }

// persistInternals mirrors PersistInternals. Every field is guarded by
// jsThread.
type persistInternals struct {
	name    string
	storage Storage
	clock   xs.Clock

	isEvent           bool
	hasPendingContext bool
	pendingContext    any
	hasPendingEvents  bool
	pendingEvents     []xs.Event
	pendingCheckpoint any
	hasFlushTimeout   bool
	flushTimeoutID    xs.TimerID
	// flushTimeoutGen identifies the scheduled timeout so a callback that
	// fired before ClearTimeout (real clock) does nothing.
	flushTimeoutGen int
	pendingWrite    *Promise[struct{}]

	lastScheduledRevision int
	// clearEpoch is incremented by ClearStorage to invalidate pending writes
	// and reads.
	clearEpoch int

	writeSnapshot func(ctx any) *Promise[struct{}]
	writeEvents   func(events []xs.Event, checkpoint any) *Promise[struct{}]
}

// realClock schedules timeouts with time.AfterFunc.
type realClock struct{}

func (realClock) SetTimeout(fn func(), d time.Duration) xs.TimerID { return time.AfterFunc(d, fn) }
func (realClock) ClearTimeout(id xs.TimerID) {
	if t, ok := id.(*time.Timer); ok {
		t.Stop()
	}
}

// noopStorage mirrors noopStorage.
var noopStorage Storage = StorageFuncs{}

// flush mirrors internals.flush(). Caller holds jsThread.
func (in *persistInternals) flush() *Promise[struct{}] {
	if in.hasFlushTimeout {
		in.clock.ClearTimeout(in.flushTimeoutID)
		in.hasFlushTimeout = false
		in.flushTimeoutID = nil
	}
	if in.isEvent {
		if in.hasPendingEvents {
			events, checkpoint := in.pendingEvents, in.pendingCheckpoint
			in.hasPendingEvents, in.pendingEvents, in.pendingCheckpoint = false, nil, nil
			return in.writeEvents(events, checkpoint)
		}
	} else if in.hasPendingContext {
		ctx := in.pendingContext
		in.hasPendingContext, in.pendingContext = false, nil
		return in.writeSnapshot(ctx)
	}
	return in.pendingWrite
}

// scheduleFlush mirrors `setTimeout(() => void internals.flush(), throttle)`.
func (in *persistInternals) scheduleFlush(throttle time.Duration) {
	if in.hasFlushTimeout {
		return
	}
	in.flushTimeoutGen++
	gen := in.flushTimeoutGen
	in.hasFlushTimeout = true
	in.flushTimeoutID = in.clock.SetTimeout(func() {
		defer jsThread.enter()()
		if !in.hasFlushTimeout || in.flushTimeoutGen != gen {
			return
		}
		in.hasFlushTimeout = false
		in.flushTimeoutID = nil
		in.flush()
	}, throttle)
}

// enqueueWrite mirrors enqueueWrite(internals, write): writes run in order;
// a write starts after the previous asynchronous write settled (fulfilled or
// rejected). Caller holds jsThread.
func (in *persistInternals) enqueueWrite(write func() *Promise[struct{}]) *Promise[struct{}] {
	var result *Promise[struct{}]
	if prev := in.pendingWrite; prev != nil {
		// pendingWrite.then(write, write)
		result = promiseGo(func() (struct{}, error) {
			<-prev.Done()
			var p *Promise[struct{}]
			func() {
				defer jsThread.enter()()
				p = write()
			}()
			return p.Wait()
		})
	} else {
		result = write()
	}
	if result == nil {
		return nil
	}
	// result.finally(() => { if (pendingWrite === pending) pendingWrite = null })
	pending, resolve, reject := NewPromise[struct{}]()
	go func() {
		_, err := result.Wait()
		func() {
			defer jsThread.enter()()
			if in.pendingWrite == pending {
				in.pendingWrite = nil
			}
		}()
		if err != nil {
			reject(err)
		} else {
			resolve(struct{}{})
		}
	}()
	in.pendingWrite = pending
	return pending
}

// writeToStorage mirrors the body shared by writeSnapshotToStorage and
// writeEventsToStorage: serialize, setItem, then onDone (sync or after the
// returned promise) or onError.
func writeToStorage(in *persistInternals, serialize func() string, done any, onDone func(any), onError func(any)) *Promise[struct{}] {
	callDone := func() {
		if onDone != nil {
			onDone(done)
		}
	}
	return in.enqueueWrite(func() *Promise[struct{}] {
		var result *Promise[struct{}]
		func() {
			defer func() {
				if r := recover(); r != nil && onError != nil {
					onError(r)
				}
			}()
			result = in.storage.SetItem(in.name, serialize())
			if result == nil {
				callDone()
			}
		}()
		if result == nil {
			return nil
		}
		// result.then(() => onDone(...)).catch((err) => onError(err)); a
		// panicking onError rejects the returned promise.
		return promiseGo(func() (struct{}, error) {
			var failure any
			if _, err := result.Wait(); err != nil {
				failure = thrown(err)
			}
			defer jsThread.enter()()
			if failure == nil {
				failure = recoverValue(callDone)
			}
			if failure != nil && onError != nil {
				onError(failure)
			}
			return struct{}{}, nil
		})
	})
}

// recoverValue runs fn and returns the value it panicked with (nil if none).
func recoverValue(fn func()) (r any) {
	defer func() { r = recover() }()
	fn()
	return nil
}

// ---- JSON helpers ----

// jsonDecode mirrors JSON.parse: objects are map[string]any, arrays []any,
// integral numbers int and other numbers float64.
func jsonDecode(s string) any {
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		panic(fmt.Errorf("SyntaxError: %w", err))
	}
	return normalizeJSON(v)
}

func normalizeJSON(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil && i >= math.MinInt && i <= math.MaxInt {
			return int(i)
		}
		f, _ := t.Float64()
		return f
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeJSON(e)
		}
	case []any:
		for i, e := range t {
			t[i] = normalizeJSON(e)
		}
	}
	return v
}

func jsonEncode(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("TypeError: %w", err))
	}
	return string(b)
}

// toJSONObject converts a value to a fresh JSON object (nil when it is not
// an object); a map input is copied, never shared.
func toJSONObject(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return maps.Clone(m)
	}
	m, _ := jsonDecode(jsonEncode(v)).(map[string]any)
	return m
}

// fromJSON converts a JSON-able value into C.
func fromJSON[C any](v any) C {
	if c, ok := v.(C); ok {
		return c
	}
	var out C
	if err := json.Unmarshal([]byte(jsonEncode(v)), &out); err != nil {
		panic(fmt.Errorf("TypeError: %w", err))
	}
	return out
}

// versionsEqual mirrors `stored.version !== currentVersion` (negated) for
// JSON numbers and strings.
func versionsEqual(a, b any) bool {
	fa, aNum := toFloat(a)
	fb, bNum := toFloat(b)
	if aNum || bNum {
		return aNum && bNum && fa == fb
	}
	return objectIs(a, b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func (o PersistOptions[C]) version() any {
	if o.Version == nil {
		return 0
	}
	return o.Version
}

func (o PersistOptions[C]) onError(err any) {
	if o.OnError != nil {
		o.OnError(err)
	}
}

func (o PersistOptions[C]) serializeSnapshot(v PersistStorageValue) string {
	if o.Serialize != nil {
		return o.Serialize(v)
	}
	return jsonEncode(v)
}

func (o PersistOptions[C]) deserializeSnapshot(s string) PersistStorageValue {
	if o.Deserialize != nil {
		return o.Deserialize(s)
	}
	m, ok := jsonDecode(s).(map[string]any)
	if !ok {
		panic(errors.New("TypeError: persisted value is not an object"))
	}
	return PersistStorageValue{Context: m["context"], Version: m["version"]}
}

func (o PersistOptions[C]) serializeEvents(v PersistEventStorageValue) string {
	if o.SerializeEvents != nil {
		return o.SerializeEvents(v)
	}
	return jsonEncode(v)
}

func (o PersistOptions[C]) deserializeEvents(s string) PersistEventStorageValue {
	if o.DeserializeEvents != nil {
		return o.DeserializeEvents(s)
	}
	m, ok := jsonDecode(s).(map[string]any)
	if !ok {
		panic(errors.New("TypeError: persisted value is not an object"))
	}
	value := PersistEventStorageValue{Version: m["version"], Checkpoint: m["checkpoint"]}
	raw, _ := m["events"].([]any)
	for _, e := range raw {
		obj, _ := e.(map[string]any)
		value.Events = append(value.Events, xs.E(obj))
	}
	return value
}

// mergeContext mirrors mergeContext: options.merge or
// `{ ...currentContext, ...persistedContext }`.
func (o PersistOptions[C]) mergeContext(persisted any, current C) C {
	if o.Merge != nil {
		return o.Merge(persisted, current)
	}
	merged := toJSONObject(current)
	if merged == nil {
		merged = map[string]any{}
	}
	maps.Copy(merged, toJSONObject(persisted))
	return fromJSON[C](merged)
}

// migrateSnapshot mirrors migrateSnapshotIfNeeded.
func (o PersistOptions[C]) migrateSnapshot(stored PersistStorageValue) any {
	if !versionsEqual(stored.Version, o.version()) && o.Migrate != nil {
		return o.Migrate(stored.Context, stored.Version)
	}
	return stored.Context
}

// migrateEvents mirrors migrateEventsIfNeeded.
func (o PersistOptions[C]) migrateEvents(stored PersistEventStorageValue) []xs.Event {
	if !versionsEqual(stored.Version, o.version()) && o.MigrateEvents != nil {
		return o.MigrateEvents(stored.Events, stored.Version)
	}
	return stored.Events
}

// createInternals mirrors createInternals(options).
func createInternals[C any](options PersistOptions[C]) *persistInternals {
	in := &persistInternals{
		name:    options.Name,
		storage: options.Storage,
		clock:   options.Clock,
		isEvent: options.Strategy == PersistEvent,
	}
	if in.storage == nil {
		in.storage = CreateJSONStorage(func() Storage { return noopStorage })
	}
	if in.clock == nil {
		in.clock = realClock{}
	}
	in.writeSnapshot = func(ctxAny any) *Promise[struct{}] {
		ctx := ctxAny.(C)
		var toPersist any = ctx
		if options.Pick != nil {
			toPersist = options.Pick(ctx)
		}
		value := PersistStorageValue{Context: toPersist, Version: options.version()}
		return writeToStorage(in, func() string { return options.serializeSnapshot(value) },
			toPersist, options.OnDone, options.OnError)
	}
	in.writeEvents = func(events []xs.Event, checkpoint any) *Promise[struct{}] {
		value := PersistEventStorageValue{Events: events, Version: options.version(), Checkpoint: checkpoint}
		return writeToStorage(in, func() string { return options.serializeEvents(value) },
			events, options.OnDone, options.OnError)
	}
	return in
}

// reportAsyncReadError mirrors `void storedValue.catch((error) => onError(error))`.
func reportAsyncReadError(p *Promise[*string], onError func(any)) {
	go func() {
		if _, err := p.Wait(); err != nil && onError != nil {
			defer jsThread.enter()()
			defer func() { _ = recover() }()
			onError(thrown(err))
		}
	}()
}

// rehydrateState extracts `event.state` when `event.clearEpoch` matches.
func rehydrateState(event xs.Event, clearEpoch int) string {
	e, _ := event.(xs.E)
	if e == nil {
		return ""
	}
	if epoch, _ := e["clearEpoch"].(int); epoch != clearEpoch {
		return ""
	}
	if s, ok := e["state"].(*string); ok && s != nil {
		return *s
	}
	return ""
}

func persistedStatus[C any](s *StoreSnapshot[C]) (persistStatus, bool) {
	st, ok := s.ext(persistKeyStatus).(persistStatus)
	return st, ok
}

func hydratedStatus[C any](s *StoreSnapshot[C]) persistStatus {
	st, _ := persistedStatus(s)
	st.Hydrated = true
	return st
}

func revisionOf[C any](s *StoreSnapshot[C]) int {
	r, _ := s.ext(persistKeyRevision).(int)
	return r
}

// persistSnapshotFromLogic mirrors persistSnapshotFromLogic.
func persistSnapshotFromLogic[C any](logic StoreLogic[C], options PersistOptions[C]) StoreLogic[C] {
	assertNoInternalEventTypeCollisions(logic.EventTypes, []string{persistRehydrateEventType}, "persist")
	in := createInternals(options)
	throttle := options.Throttle

	enhanced := logic
	enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] {
		base := logic.GetInitialSnapshot()
		withMeta := func(s *StoreSnapshot[C], hydrated bool) *StoreSnapshot[C] {
			return s.withExt(persistKeyStatus, persistStatus{Hydrated: hydrated}, persistKeyInternals, in)
		}
		if options.SkipHydration {
			return withMeta(base, false)
		}
		var result *StoreSnapshot[C]
		func() {
			defer func() {
				if r := recover(); r != nil {
					options.onError(r)
					result = withMeta(base, true)
				}
			}()
			stored, promise := in.storage.GetItem(in.name)
			if promise != nil {
				// Async storage — can't hydrate synchronously
				reportAsyncReadError(promise, options.OnError)
				result = withMeta(base, false)
				return
			}
			if stored == nil {
				result = withMeta(base, true)
				return
			}
			parsed := options.deserializeSnapshot(*stored)
			persisted := options.migrateSnapshot(parsed)
			merged := options.mergeContext(persisted, base.Context)
			result = withMeta(base.withContext(merged), true)
		}()
		return result
	}

	enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
		if event.EventType() == persistRehydrateEventType {
			// Discard reads that started before `clearStorage`
			rawState := rehydrateState(event, in.clearEpoch)
			if rawState == "" {
				return StoreTransitionResult[C]{
					Snapshot: snapshot.withExt(persistKeyStatus, hydratedStatus(snapshot)),
					Effects:  []StoreEffect[C]{},
				}
			}
			var result *StoreSnapshot[C]
			func() {
				defer func() {
					if r := recover(); r != nil {
						options.onError(r)
						result = snapshot.withExt(persistKeyStatus, hydratedStatus(snapshot))
					}
				}()
				parsed := options.deserializeSnapshot(rawState)
				persisted := options.migrateSnapshot(parsed)
				merged := options.mergeContext(persisted, snapshot.Context)
				result = snapshot.withContext(merged).withExt(persistKeyStatus, hydratedStatus(snapshot))
			}()
			return StoreTransitionResult[C]{Snapshot: result, Effects: []StoreEffect[C]{}}
		}

		// Delegate to wrapped logic
		inner := logic.Transition(snapshot, event)
		next := inner.Snapshot

		revision := revisionOf(snapshot) + 1
		clearEpoch := in.clearEpoch

		status, ok := persistedStatus(snapshot)
		if !ok {
			status = persistStatus{Hydrated: false}
		}
		withMeta := next.withExt(
			persistKeyRevision, revision,
			persistKeyStatus, status,
			persistKeyInternals, in,
		)

		// Don't write to storage until hydrated
		if !status.Hydrated {
			return StoreTransitionResult[C]{Snapshot: withMeta, Effects: inner.Effects}
		}
		if options.Filter != nil && !options.Filter(event) {
			return StoreTransitionResult[C]{Snapshot: withMeta, Effects: inner.Effects}
		}

		// Commit before wrapped effects can trigger another event. Subscribers
		// can already have committed a newer eligible event before effects
		// begin.
		persistEffect := func(*StoreEffectEnqueue[C]) {
			defer jsThread.enter()()
			if clearEpoch != in.clearEpoch || revision <= in.lastScheduledRevision {
				return
			}
			in.lastScheduledRevision = revision
			if throttle > 0 {
				in.hasPendingContext, in.pendingContext = true, next.Context
				in.scheduleFlush(throttle)
			} else {
				in.writeSnapshot(next.Context)
			}
		}
		effects := append([]StoreEffect[C]{{Run: persistEffect}}, inner.Effects...)
		return StoreTransitionResult[C]{Snapshot: withMeta, Effects: effects}
	}
	return enhanced
}

// persistEventFromLogic mirrors persistEventFromLogic.
func persistEventFromLogic[C any](logic StoreLogic[C], options PersistOptions[C]) StoreLogic[C] {
	assertNoInternalEventTypeCollisions(logic.EventTypes, []string{persistRehydrateEventType}, "persist")
	in := createInternals(options)
	throttle := options.Throttle
	maxEvents := options.MaxEvents

	replayEvents := func(base *StoreSnapshot[C], events []xs.Event) *StoreSnapshot[C] {
		current := base
		for _, ev := range events {
			current = logic.Transition(current, ev).Snapshot
		}
		return current
	}
	fromCheckpoint := func(base *StoreSnapshot[C], checkpoint any) *StoreSnapshot[C] {
		if isTruthy(checkpoint) {
			return base.withContext(fromJSON[C](checkpoint))
		}
		return base
	}
	withMeta := func(s *StoreSnapshot[C], events []xs.Event, checkpoint any, status persistStatus) *StoreSnapshot[C] {
		return s.withExt(
			persistKeyClearEpoch, in.clearEpoch,
			persistKeyEvents, events,
			persistKeyCheckpoint, checkpoint,
			persistKeyStatus, status,
			persistKeyInternals, in,
		)
	}

	enhanced := logic
	enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] {
		base := logic.GetInitialSnapshot()
		if options.SkipHydration {
			return withMeta(base, []xs.Event{}, nil, persistStatus{Hydrated: false})
		}
		var result *StoreSnapshot[C]
		func() {
			defer func() {
				if r := recover(); r != nil {
					options.onError(r)
					result = withMeta(base, []xs.Event{}, nil, persistStatus{Hydrated: true})
				}
			}()
			stored, promise := in.storage.GetItem(in.name)
			if promise != nil {
				reportAsyncReadError(promise, options.OnError)
				result = withMeta(base, []xs.Event{}, nil, persistStatus{Hydrated: false})
				return
			}
			if stored == nil {
				result = withMeta(base, []xs.Event{}, nil, persistStatus{Hydrated: true})
				return
			}
			parsed := options.deserializeEvents(*stored)
			events := options.migrateEvents(parsed)
			replayed := replayEvents(fromCheckpoint(base, parsed.Checkpoint), events)
			result = withMeta(replayed, events, parsed.Checkpoint, persistStatus{Hydrated: true})
		}()
		return result
	}

	enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
		clearEpoch := in.clearEpoch
		epoch, _ := snapshot.ext(persistKeyClearEpoch).(int)
		historyCleared := epoch != clearEpoch
		var prevEvents []xs.Event
		var prevCheckpoint any
		if historyCleared {
			prevEvents = []xs.Event{}
			prevCheckpoint = snapshot.Context
		} else {
			prevEvents, _ = snapshot.ext(persistKeyEvents).([]xs.Event)
			if prevEvents == nil {
				prevEvents = []xs.Event{}
			}
			prevCheckpoint = snapshot.ext(persistKeyCheckpoint)
		}

		if event.EventType() == persistRehydrateEventType {
			// Discard reads that started before `clearStorage`
			rawState := rehydrateState(event, clearEpoch)
			keep := func() *StoreSnapshot[C] {
				return snapshot.withExt(
					persistKeyClearEpoch, clearEpoch,
					persistKeyEvents, prevEvents,
					persistKeyCheckpoint, prevCheckpoint,
					persistKeyStatus, hydratedStatus(snapshot),
				)
			}
			if rawState == "" {
				return StoreTransitionResult[C]{Snapshot: keep(), Effects: []StoreEffect[C]{}}
			}
			var result *StoreSnapshot[C]
			func() {
				defer func() {
					if r := recover(); r != nil {
						options.onError(r)
						result = keep()
					}
				}()
				parsed := options.deserializeEvents(rawState)
				events := options.migrateEvents(parsed)
				replayed := replayEvents(fromCheckpoint(logic.GetInitialSnapshot(), parsed.Checkpoint), events)
				result = replayed.withExt(
					persistKeyRevision, snapshot.ext(persistKeyRevision),
					persistKeyClearEpoch, clearEpoch,
					persistKeyEvents, events,
					persistKeyCheckpoint, parsed.Checkpoint,
					persistKeyStatus, hydratedStatus(snapshot),
					persistKeyInternals, in,
				)
			}()
			return StoreTransitionResult[C]{Snapshot: result, Effects: []StoreEffect[C]{}}
		}

		// Delegate to wrapped logic
		inner := logic.Transition(snapshot, event)
		revision := revisionOf(snapshot) + 1

		status, ok := persistedStatus(snapshot)
		if !ok {
			status = persistStatus{Hydrated: false}
		}
		withMetaSnap := inner.Snapshot.withExt(
			persistKeyRevision, revision,
			persistKeyClearEpoch, clearEpoch,
			persistKeyEvents, prevEvents,
			persistKeyCheckpoint, prevCheckpoint,
			persistKeyStatus, status,
			persistKeyInternals, in,
		)

		// Don't write to storage until hydrated
		if !status.Hydrated {
			return StoreTransitionResult[C]{Snapshot: withMetaSnap, Effects: inner.Effects}
		}

		// Append event to persisted list
		nextEvents := append(append(make([]xs.Event, 0, len(prevEvents)+1), prevEvents...), event)
		nextCheckpoint := prevCheckpoint
		if maxEvents > 0 && len(nextEvents) > maxEvents {
			// Compute checkpoint by replaying dropped events from previous
			// checkpoint
			dropped := nextEvents[:len(nextEvents)-maxEvents]
			checkpointSnapshot := replayEvents(fromCheckpoint(logic.GetInitialSnapshot(), prevCheckpoint), dropped)
			nextCheckpoint = checkpointSnapshot.Context
			nextEvents = nextEvents[len(nextEvents)-maxEvents:]
		}

		withEvents := withMetaSnap.withExt(persistKeyEvents, nextEvents, persistKeyCheckpoint, nextCheckpoint)

		persistEffect := func(*StoreEffectEnqueue[C]) {
			defer jsThread.enter()()
			if clearEpoch != in.clearEpoch || revision <= in.lastScheduledRevision {
				return
			}
			in.lastScheduledRevision = revision
			if throttle > 0 {
				in.hasPendingEvents, in.pendingEvents, in.pendingCheckpoint = true, nextEvents, nextCheckpoint
				in.scheduleFlush(throttle)
			} else {
				in.writeEvents(nextEvents, nextCheckpoint)
			}
		}
		effects := append([]StoreEffect[C]{{Run: persistEffect}}, inner.Effects...)
		return StoreTransitionResult[C]{Snapshot: withEvents, Effects: effects}
	}
	return enhanced
}

// isTruthy mirrors JS truthiness for persisted JSON values.
func isTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case int:
		return t != 0
	case float64:
		return t != 0 && !math.IsNaN(t)
	}
	return true
}

// Persist mirrors persist(options) from @xstate/store/persist.
func Persist[C any](opts PersistOptions[C]) StoreExtension[C] {
	return func(logic StoreLogic[C]) StoreLogic[C] {
		if opts.Strategy == PersistEvent {
			return persistEventFromLogic(logic, opts)
		}
		return persistSnapshotFromLogic(logic, opts)
	}
}

// jsonStorage is the adapter returned by CreateJSONStorage.
type jsonStorage struct{ storage Storage }

func (j jsonStorage) GetItem(name string) (v *string, p *Promise[*string]) {
	defer func() {
		if recover() != nil {
			v, p = nil, nil
		}
	}()
	v, p = j.storage.GetItem(name)
	if p == nil {
		return v, nil
	}
	// result.catch(() => null)
	inner := p
	return nil, promiseGo(func() (*string, error) {
		value, err := inner.Wait()
		if err != nil {
			return nil, nil
		}
		return value, nil
	})
}

func (j jsonStorage) SetItem(name, value string) (p *Promise[struct{}]) {
	defer func() {
		if recover() != nil {
			p = nil
		}
	}()
	return swallowRejection(j.storage.SetItem(name, value))
}

func (j jsonStorage) RemoveItem(name string) (p *Promise[struct{}]) {
	defer func() {
		if recover() != nil {
			p = nil
		}
	}()
	return swallowRejection(j.storage.RemoveItem(name))
}

// swallowRejection mirrors `result.catch(() => {})`.
func swallowRejection(p *Promise[struct{}]) *Promise[struct{}] {
	if p == nil {
		return nil
	}
	return promiseGo(func() (struct{}, error) {
		p.Wait()
		return struct{}{}, nil
	})
}

// CreateJSONStorage mirrors createJSONStorage(getStorage): panics from
// getStorage yield a no-op storage; panics/rejections of the wrapped
// storage are swallowed (GetItem → null).
func CreateJSONStorage(getStorage func() Storage) (s Storage) {
	defer func() {
		if recover() != nil {
			s = noopStorage
		}
	}()
	storage := getStorage()
	if storage == nil {
		return noopStorage
	}
	return jsonStorage{storage: storage}
}

// internalsOf reads the persist internals of a store's current snapshot.
func internalsOf[C any](s *Store[C], fn string) *persistInternals {
	in, _ := s.currentSnapshot.ext(persistKeyInternals).(*persistInternals)
	if in == nil {
		panic(fmt.Errorf("%s: store does not have a persist extension", fn))
	}
	return in
}

// ClearStorage mirrors clearStorage(store). nil means the removal completed
// synchronously. Panics when the store has no persist extension.
func ClearStorage[C any](s *Store[C]) *Promise[struct{}] {
	defer jsThread.enter()()
	in := internalsOf(s, "clearStorage")
	if in.hasFlushTimeout {
		in.clock.ClearTimeout(in.flushTimeoutID)
		in.hasFlushTimeout, in.flushTimeoutID = false, nil
	}
	in.hasPendingContext, in.pendingContext = false, nil
	in.hasPendingEvents, in.pendingEvents, in.pendingCheckpoint = false, nil, nil
	// Event history resets on the next transition, keeping existing
	// snapshots immutable.
	in.clearEpoch++
	return in.enqueueWrite(func() *Promise[struct{}] { return in.storage.RemoveItem(in.name) })
}

// FlushStorage mirrors flushStorage(store). nil means everything was written
// synchronously. Panics when the store has no persist extension.
func FlushStorage[C any](s *Store[C]) *Promise[struct{}] {
	defer jsThread.enter()()
	return drain(internalsOf(s, "flushStorage"))
}

// drain flushes until no write is pending: callbacks such as onDone may send
// events that queue more writes. Caller holds jsThread.
func drain(in *persistInternals) *Promise[struct{}] {
	result := in.flush()
	if result == nil {
		return nil
	}
	return promiseGo(func() (struct{}, error) {
		for result != nil {
			if _, err := result.Wait(); err != nil {
				return struct{}{}, err
			}
			func() {
				defer jsThread.enter()()
				result = in.flush()
			}()
		}
		return struct{}{}, nil
	})
}

// IsHydrated mirrors isHydrated(store).
func IsHydrated[C any](s *Store[C]) bool {
	st, _ := persistedStatus(s.GetSnapshot())
	return st.Hydrated
}

// RehydrateStore mirrors rehydrateStore(store), an async function: it
// always returns a promise, which rejects when the store has no persist
// extension or the storage read fails.
func RehydrateStore[C any](s *Store[C]) *Promise[struct{}] {
	// The synchronous part of the JS async function: read the internals,
	// capture the clear epoch and start the read.
	var (
		clearEpoch int
		value      *string
		promise    *Promise[*string]
	)
	failure := recoverValue(func() {
		defer jsThread.enter()()
		in := internalsOf(s, "rehydrateStore")
		clearEpoch = in.clearEpoch
		value, promise = in.storage.GetItem(in.name)
	})
	return promiseGo(func() (struct{}, error) {
		if failure != nil {
			return struct{}{}, panicError(failure)
		}
		data := value
		if promise != nil {
			v, err := promise.Wait()
			if err != nil {
				return struct{}{}, err
			}
			data = v
		}
		s.Send(xs.E{"type": persistRehydrateEventType, "state": data, "clearEpoch": clearEpoch})
		return struct{}{}, nil
	})
}

// BroadcastMessage mirrors the `{ type: 'xstate-store-update', name }`
// message posted by broadcast storage.
type BroadcastMessage struct {
	Type string
	Name string
}

// BroadcastChannel mirrors the subset of the DOM BroadcastChannel used by
// broadcast storage. Messages posted on a channel reach every other channel
// with the same name (not the sender).
type BroadcastChannel interface {
	PostMessage(data any)
	// OnMessage mirrors addEventListener('message', fn); Unsubscribe mirrors
	// removeEventListener.
	OnMessage(fn func(data any)) xs.Subscription
	Close()
}

// BroadcastStorageOptions mirrors BroadcastStorageOptions.
type BroadcastStorageOptions struct {
	// Channel mirrors `channel`; "" means "xstate-store".
	Channel string
	// NewChannel mirrors the global `new BroadcastChannel(name)`. nil mirrors
	// an environment without BroadcastChannel: CreateBroadcastStorage panics
	// with "createBroadcastStorage: BroadcastChannel is not available in this
	// environment".
	NewChannel func(name string) BroadcastChannel
}

// BroadcastStorage mirrors the storage returned by createBroadcastStorage.
type BroadcastStorage struct {
	base    Storage
	channel BroadcastChannel
}

func (b *BroadcastStorage) GetItem(name string) (*string, *Promise[*string]) {
	return b.base.GetItem(name)
}

func (b *BroadcastStorage) SetItem(name, value string) *Promise[struct{}] {
	result := b.base.SetItem(name, value)
	broadcast := func() {
		b.channel.PostMessage(BroadcastMessage{Type: "xstate-store-update", Name: name})
	}
	if result != nil {
		// result.then(broadcast)
		return promiseGo(func() (struct{}, error) {
			if _, err := result.Wait(); err != nil {
				return struct{}{}, err
			}
			broadcast()
			return struct{}{}, nil
		})
	}
	broadcast()
	return nil
}

func (b *BroadcastStorage) RemoveItem(name string) *Promise[struct{}] { return b.base.RemoveItem(name) }

// Channel mirrors broadcastStorage.channel.
func (b *BroadcastStorage) Channel() BroadcastChannel { return b.channel }

// CreateBroadcastStorage mirrors createBroadcastStorage(baseStorage, options?).
func CreateBroadcastStorage(base Storage, opts ...BroadcastStorageOptions) *BroadcastStorage {
	var options BroadcastStorageOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.NewChannel == nil {
		panic(errors.New("createBroadcastStorage: BroadcastChannel is not available in this environment"))
	}
	name := options.Channel
	if name == "" {
		name = "xstate-store"
	}
	return &BroadcastStorage{base: base, channel: options.NewChannel(name)}
}

// SubscribeToBroadcastStorage mirrors subscribeToBroadcastStorage(store):
// returns the unsubscribe function. Panics when the store has no persist
// extension or its storage is not a *BroadcastStorage.
func SubscribeToBroadcastStorage[C any](s *Store[C]) func() {
	in := func() *persistInternals {
		defer jsThread.enter()()
		return internalsOf(s, "subscribeToBroadcastStorage")
	}()
	bs, ok := in.storage.(*BroadcastStorage)
	if !ok || bs.channel == nil {
		panic(errors.New("subscribeToBroadcastStorage: store storage must be wrapped with createBroadcastStorage()"))
	}
	storeName := in.name
	sub := bs.channel.OnMessage(func(data any) {
		var msg BroadcastMessage
		switch m := data.(type) {
		case BroadcastMessage:
			msg = m
		case *BroadcastMessage:
			if m == nil {
				return
			}
			msg = *m
		default:
			return
		}
		if msg.Type == "xstate-store-update" && msg.Name == storeName {
			RehydrateStore(s)
		}
	})
	return sub.Unsubscribe
}
