package graph

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestModel mirrors TestModel: an abstract model of a system under test used
// to generate test paths that verify the model's states are reachable in the
// system under test.
type TestModel[S xs.Snapshot] struct {
	TestLogic               xs.TypedActorLogic[S]
	Options                 TestModelOptions[S]
	DefaultTraversalOptions *TraversalOptions[S]
}

// NewTestModel mirrors `new TestModel(logic, options)`.
func NewTestModel[S xs.Snapshot](logic xs.TypedActorLogic[S], options ...TestModelOptions[S]) *TestModel[S] {
	m := &TestModel[S]{TestLogic: logic}
	m.Options = m.GetDefaultOptions()
	if len(options) > 0 {
		m.Options = m.Options.overlay(options[0])
	}
	return m
}

// CreateTestModel mirrors createTestModel(machine, options). It panics like
// ValidateMachine for machines with invocations, delays or delayed actions.
func CreateTestModel[C any](machine *xs.StateMachine[C], options ...TestModelOptions[*xs.MachineSnapshot[C]]) *TestModel[*xs.MachineSnapshot[C]] {
	ValidateMachine(machine)

	var opts TestModelOptions[*xs.MachineSnapshot[C]]
	if len(options) > 0 {
		opts = options[0]
	}
	serializeEvent := opts.SerializeEvent
	if serializeEvent == nil {
		serializeEvent = jsonStringifyEvent
	}
	serializeTransition := opts.SerializeTransition
	if serializeTransition == nil {
		serializeTransition = func(state *xs.MachineSnapshot[C], event xs.Event, prevState *xs.MachineSnapshot[C]) string {
			return serializeMachineTransition(state, event, prevState, serializeEvent)
		}
	}
	getEvents := opts.TraversalOptions
	otherOptions := opts
	otherOptions.TraversalOptions = withoutEvents(opts.TraversalOptions)

	return NewTestModel[*xs.MachineSnapshot[C]](machine, TestModelOptions[*xs.MachineSnapshot[C]]{
		TraversalOptions: TraversalOptions[*xs.MachineSnapshot[C]]{
			SerializeState: func(state *xs.MachineSnapshot[C], event xs.Event, prevState *xs.MachineSnapshot[C]) string {
				// Only consider the `state` if `serializeTransition()` is opted out (empty string)
				return SerializeSnapshot(state) + serializeTransition(state, event, prevState)
			},
			EventsFn: func(state *xs.MachineSnapshot[C]) []xs.Event {
				return eventsWithOwnDescriptors(state, getEvents.events(state))
			},
		},
		StateMatcher: func(state *xs.MachineSnapshot[C], key string) bool {
			if strings.HasPrefix(key, "#") {
				return slices.Contains(state.StateNodes(), machine.GetStateNodeByID(key))
			}
			return state.Matches(key)
		},
	}.overlay(otherOptions))
}

// GetDefaultOptions mirrors testModel.getDefaultOptions().
func (m *TestModel[S]) GetDefaultOptions() TestModelOptions[S] {
	return TestModelOptions[S]{
		TraversalOptions: TraversalOptions[S]{
			SerializeState: func(state S, _ xs.Event, _ S) string { return jsonStringify(state) },
			SerializeEvent: jsonStringifyEvent,
			Events:         []xs.Event{},
		},
		// For non-state-machine test models, we cannot identify
		// separate transitions, so just use event type
		SerializeTransition: func(state S, event xs.Event, _ S) string {
			eventType := "undefined" // `${event?.type}`
			if event != nil {
				eventType = event.EventType()
			}
			return jsonStringify(state) + "|" + eventType
		},
		StateMatcher: func(_ S, stateKey string) bool { return stateKey == "*" },
		Logger: TestModelLogger{
			Log:   func(msg string) { fmt.Fprintln(os.Stdout, msg) },
			Error: func(msg string) { fmt.Fprintln(os.Stderr, msg) },
		},
	}
}

// GetPaths mirrors testModel.getPaths(pathGenerator, options).
func (m *TestModel[S]) GetPaths(pathGenerator PathGenerator[S], options ...GetPathOptions[S]) []TestPath[S] {
	opts := firstGetPathOptions(options)
	paths := pathGenerator(m.TestLogic, m.resolveOptions(&TestModelOptions[S]{TraversalOptions: opts.TraversalOptions}).TraversalOptions)
	if !opts.AllowDuplicatePaths {
		paths = deduplicatePaths(paths, nil)
	}
	return m.toTestPaths(paths)
}

// GetShortestPaths mirrors testModel.getShortestPaths(options).
func (m *TestModel[S]) GetShortestPaths(options ...GetPathOptions[S]) []TestPath[S] {
	return m.GetPaths(CreateShortestPathsGen[S](), options...)
}

// GetShortestPathsFrom mirrors testModel.getShortestPathsFrom(paths, options).
func (m *TestModel[S]) GetShortestPathsFrom(paths []TestPath[S], options ...GetPathOptions[S]) []TestPath[S] {
	return m.getPathsFrom(m.GetShortestPaths, paths, options)
}

// GetSimplePaths mirrors testModel.getSimplePaths(options).
func (m *TestModel[S]) GetSimplePaths(options ...GetPathOptions[S]) []TestPath[S] {
	return m.GetPaths(CreateSimplePathsGen[S](), options...)
}

// GetSimplePathsFrom mirrors testModel.getSimplePathsFrom(paths, options).
func (m *TestModel[S]) GetSimplePathsFrom(paths []TestPath[S], options ...GetPathOptions[S]) []TestPath[S] {
	return m.getPathsFrom(m.GetSimplePaths, paths, options)
}

// getPathsFrom is the shared body of getShortestPathsFrom and
// getSimplePathsFrom: continue every path with the paths from its state.
func (m *TestModel[S]) getPathsFrom(getPaths func(...GetPathOptions[S]) []TestPath[S], paths []TestPath[S], options []GetPathOptions[S]) []TestPath[S] {
	resultPaths := []TestPath[S]{}

	for _, path := range paths {
		opts := firstGetPathOptions(options)
		opts.FromState = path.State
		for _, tailPath := range getPaths(opts) {
			resultPaths = append(resultPaths, m.toTestPath(JoinPaths(path.StatePath, tailPath.StatePath)))
		}
	}

	return resultPaths
}

func firstGetPathOptions[S xs.Snapshot](options []GetPathOptions[S]) GetPathOptions[S] {
	if len(options) > 0 {
		return options[0]
	}
	return GetPathOptions[S]{}
}

func (m *TestModel[S]) toTestPaths(paths []StatePath[S]) []TestPath[S] {
	out := make([]TestPath[S], 0, len(paths))
	for _, p := range paths {
		out = append(out, m.toTestPath(p))
	}
	return out
}

// toTestPath mirrors testModel._toTestPath(statePath).
func (m *TestModel[S]) toTestPath(statePath StatePath[S]) TestPath[S] {
	events := make([]string, 0, len(statePath.Steps))
	for _, s := range statePath.Steps {
		events = append(events, formatEvent(s.Event))
	}
	eventsString := strings.Join(events, " → ")

	var description string
	if view, ok := any(statePath.State).(machineSnapshotView); ok && xs.IsMachineSnapshot(statePath.State) {
		description = "Reaches " + strings.TrimSpace(getDescription(view)) + ": " + eventsString
	} else {
		description = jsonStringify(statePath.State)
	}

	return TestPath[S]{
		StatePath: statePath,
		Test: func(params TestParam[S]) (TestPathResult[S], error) {
			return m.TestPath(statePath, params)
		},
		Description: description,
	}
}

// GetPathsFromEvents mirrors testModel.getPathsFromEvents(events, options).
func (m *TestModel[S]) GetPathsFromEvents(events []xs.Event, options ...GetPathOptions[S]) []TestPath[S] {
	var traversalOptions []TraversalOptions[S]
	if len(options) > 0 {
		traversalOptions = append(traversalOptions, options[0].TraversalOptions)
	}
	return m.toTestPaths(GetPathsFromEvents(m.TestLogic, events, traversalOptions...))
}

// GetAdjacencyMap mirrors testModel.getAdjacencyMap().
func (m *TestModel[S]) GetAdjacencyMap() AdjacencyMap[S] {
	return GetAdjacencyMap(m.TestLogic, m.Options.TraversalOptions)
}

// pathTestError mirrors the JS error whose message testPath extends with the
// formatted path trace.
type pathTestError struct {
	err   error
	trace string
}

func (e *pathTestError) Error() string { return e.err.Error() + e.trace }
func (e *pathTestError) Unwrap() error { return e.err }

// TestPath mirrors testModel.testPath(path, params, options). On failure the
// returned error's message is the original message followed by the
// formatted path trace (JS appends it to err.message).
func (m *TestModel[S]) TestPath(path StatePath[S], params TestParam[S], options ...TestModelOptions[S]) (TestPathResult[S], error) {
	testPathResult := TestPathResult[S]{}

	fail := func(err error) (TestPathResult[S], error) {
		trace := formatPathTestResult(path, testPathResult, m.Options.SerializeState, m.Options.SerializeEvent)
		return testPathResult, &pathTestError{err: err, trace: trace}
	}

	for _, step := range path.Steps {
		testPathResult.Steps = append(testPathResult.Steps, TestStepResult[S]{Step: step})
		testStepResult := &testPathResult.Steps[len(testPathResult.Steps)-1]

		if err := m.TestTransition(params, step); err != nil {
			testStepResult.Event.Error = err
			return fail(err)
		}

		if err := m.TestState(params, step.State, options...); err != nil {
			testStepResult.State.Error = err
			return fail(err)
		}
	}

	return testPathResult, nil
}

// TestState mirrors testModel.testState(params, state, options).
func (m *TestModel[S]) TestState(params TestParam[S], state S, options ...TestModelOptions[S]) error {
	var opts *TestModelOptions[S]
	if len(options) > 0 {
		opts = &options[0]
	}
	resolvedOptions := m.resolveOptions(opts)

	for _, stateTestKey := range getStateTestKeys(params, state, resolvedOptions) {
		if err := params.States[stateTestKey](state); err != nil {
			return err
		}
	}
	return nil
}

// getStateTestKeys mirrors testModel._getStateTestKeys(params, state,
// resolvedOptions). Keys are visited in sorted order.
func getStateTestKeys[S xs.Snapshot](params TestParam[S], state S, resolvedOptions TestModelOptions[S]) []string {
	keys := make([]string, 0, len(params.States))
	for k := range params.States {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	stateTestKeys := []string{}
	for _, stateKey := range keys {
		if resolvedOptions.StateMatcher(state, stateKey) {
			stateTestKeys = append(stateTestKeys, stateKey)
		}
	}

	// Fallthrough state tests
	if _, ok := params.States["*"]; len(stateTestKeys) == 0 && ok {
		stateTestKeys = append(stateTestKeys, "*")
	}

	return stateTestKeys
}

// TestTransition mirrors testModel.testTransition(params, step).
func (m *TestModel[S]) TestTransition(params TestParam[S], step Step[S]) error {
	eventExec := params.Events[step.Event.EventType()]
	if eventExec == nil {
		return nil
	}
	return eventExec(step)
}

// resolveOptions mirrors testModel._resolveOptions(options):
// { ...defaultTraversalOptions, ...options, ...overrides }.
func (m *TestModel[S]) resolveOptions(options *TestModelOptions[S]) TestModelOptions[S] {
	var resolved TestModelOptions[S]
	if m.DefaultTraversalOptions != nil {
		resolved.TraversalOptions = *m.DefaultTraversalOptions
	}
	resolved = resolved.overlay(m.Options)
	if options != nil {
		resolved = resolved.overlay(*options)
	}
	return resolved
}

// overlay mirrors the object spread `{ ...o, ...other }` for test model
// options (see TraversalOptions.overlay).
func (o TestModelOptions[S]) overlay(other TestModelOptions[S]) TestModelOptions[S] {
	o.TraversalOptions = o.TraversalOptions.overlay(other.TraversalOptions)
	if other.StateMatcher != nil {
		o.StateMatcher = other.StateMatcher
	}
	if other.Logger.Log != nil || other.Logger.Error != nil {
		o.Logger = other.Logger
	}
	if other.SerializeTransition != nil {
		o.SerializeTransition = other.SerializeTransition
	}
	return o
}

// stateValuesEqual mirrors stateValuesEqual(a, b).
func stateValuesEqual(a, b xs.StateValue) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	as, aIsString := a.(string)
	bs, bIsString := b.(string)
	if aIsString || bIsString {
		return aIsString && bIsString && as == bs
	}

	am, aIsMap := a.(map[string]any)
	bm, bIsMap := b.(map[string]any)
	if !aIsMap || !bIsMap {
		return reflect.DeepEqual(a, b)
	}

	if len(am) != len(bm) {
		return false
	}
	for key, av := range am {
		if !stateValuesEqual(av, bm[key]) {
			return false
		}
	}
	return true
}

// serializeMachineTransition mirrors serializeMachineTransition(snapshot,
// event, previousSnapshot, { serializeEvent }).
func serializeMachineTransition[C any](snapshot *xs.MachineSnapshot[C], event xs.Event, previousSnapshot *xs.MachineSnapshot[C], serializeEvent func(xs.Event) string) string {
	if event == nil || (previousSnapshot != nil && stateValuesEqual(previousSnapshot.Value, snapshot.Value)) {
		return ""
	}

	prevStateString := ""
	if previousSnapshot != nil {
		prevStateString = " from " + stateValueJSON(previousSnapshot.Value, rootNodeOf(previousSnapshot.StateNodes()))
	}

	return " via " + serializeEvent(event) + prevStateString
}
