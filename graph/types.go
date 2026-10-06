// Package graph mirrors XState's graph utilities (packages/core/src/graph):
// path generation (shortest/simple paths, paths from events), adjacency maps,
// directed graphs and model-based testing (TestModel).
package graph

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Step mirrors Step: the event that resulted in State, and that State.
type Step[S xs.Snapshot] struct {
	Event xs.Event
	State S
}

// StatePath mirrors StatePath.
type StatePath[S xs.Snapshot] struct {
	// State is the ending state of the path.
	State S
	// Steps is the ordered list of state-event pairs which reach State.
	Steps []Step[S]
	// Weight is the combined weight of all steps in the path.
	Weight int
}

// TraversalOptions mirrors TraversalOptions.
//
// Zero-valued fields fall back to the JS defaults: a nil func is "not
// provided", Limit 0 means no limit (JS `Infinity`), and a zero (nil)
// FromState means "start from the initial snapshot".
type TraversalOptions[S xs.Snapshot] struct {
	// Input is passed to the logic's initial snapshot.
	Input any
	// SerializeState mirrors serializeState(state, event, prevState). event is
	// nil and prevState is the zero value (nil) when JS passes undefined.
	SerializeState func(state S, event xs.Event, prevState S) string
	// SerializeEvent mirrors serializeEvent(event).
	SerializeEvent func(event xs.Event) string
	// Events mirrors the array form of `events`.
	Events []xs.Event
	// EventsFn mirrors the function form of `events`; it takes precedence
	// over Events when set.
	EventsFn func(state S) []xs.Event
	// FilterEvents mirrors filterEvents(snapshot, event).
	FilterEvents func(state S, event xs.Event) bool
	// Limit is the maximum number of traversals; 0 means no limit.
	Limit int
	// FromState is the snapshot traversal starts from.
	FromState S
	// StopWhen stops traversal past states for which it returns true.
	StopWhen func(state S) bool
	// ToState keeps only paths whose ending state satisfies it.
	ToState func(state S) bool
}

// AdjacencyTransition is one entry of AdjacencyValue.transitions.
type AdjacencyTransition[S xs.Snapshot] struct {
	Event xs.Event
	State S
}

// AdjacencyValue mirrors AdjacencyValue. JS keeps transitions in an object
// keyed by serialized event; EventKeys preserves that insertion order.
type AdjacencyValue[S xs.Snapshot] struct {
	State       S
	EventKeys   []string
	Transitions map[string]AdjacencyTransition[S]
}

// AdjacencyMap mirrors AdjacencyMap: serialized snapshot -> AdjacencyValue.
// Keys preserves the JS object insertion (traversal) order.
type AdjacencyMap[S xs.Snapshot] struct {
	Keys   []string
	Values map[string]*AdjacencyValue[S]
}

// AdjacencyEntry is one element returned by AdjacencyMapToArray.
type AdjacencyEntry[S xs.Snapshot] struct {
	State     S
	Event     xs.Event
	NextState S
}

// DirectedGraphLabel mirrors DirectedGraphLabel (and its toJSON()).
type DirectedGraphLabel struct {
	Text string `json:"text"`
}

// DirectedGraphEdge mirrors DirectedGraphEdge.
type DirectedGraphEdge struct {
	ID         string
	Source     *xs.StateNode
	Target     *xs.StateNode
	Label      DirectedGraphLabel
	Transition *xs.TransitionDefinition
}

// DirectedGraphEdgeJSON mirrors the result of DirectedGraphEdge.toJSON().
type DirectedGraphEdgeJSON struct {
	Source string             `json:"source"`
	Target string             `json:"target"`
	Label  DirectedGraphLabel `json:"label"`
}

// ToJSON mirrors edge.toJSON().
func (e *DirectedGraphEdge) ToJSON() DirectedGraphEdgeJSON {
	return DirectedGraphEdgeJSON{Source: e.Source.ID, Target: e.Target.ID, Label: e.Label}
}

// DirectedGraphNode mirrors DirectedGraphNode.
type DirectedGraphNode struct {
	ID        string
	StateNode *xs.StateNode
	Children  []*DirectedGraphNode
	// Edges are the edges representing all transitions from StateNode.
	Edges []*DirectedGraphEdge
}

// DirectedGraphNodeJSON mirrors the recursively serialized result of
// DirectedGraphNode.toJSON(). Children and Edges are never nil (JSON `[]`).
type DirectedGraphNodeJSON struct {
	ID       string                  `json:"id"`
	Children []DirectedGraphNodeJSON `json:"children"`
	Edges    []DirectedGraphEdgeJSON `json:"edges"`
}

// ToJSON mirrors node.toJSON(), recursively applied to children and edges.
func (n *DirectedGraphNode) ToJSON() DirectedGraphNodeJSON {
	children := make([]DirectedGraphNodeJSON, 0, len(n.Children))
	for _, child := range n.Children {
		children = append(children, child.ToJSON())
	}
	edges := make([]DirectedGraphEdgeJSON, 0, len(n.Edges))
	for _, edge := range n.Edges {
		edges = append(edges, edge.ToJSON())
	}
	return DirectedGraphNodeJSON{ID: n.ID, Children: children, Edges: edges}
}

// TestMeta mirrors TestMeta: the shape graph utilities read from a state's
// `meta` (getDescription uses Description / DescriptionFn).
type TestMeta[C any] struct {
	Test          func(testContext any, state *xs.MachineSnapshot[C]) error
	Description   string
	DescriptionFn func(state *xs.MachineSnapshot[C]) string
	Skip          bool
}

// describe implements metaDescriber: DescriptionFn wins over Description,
// like JS `typeof description === 'function'`.
func (m TestMeta[C]) describe(snapshot any) (text string, quoted bool) {
	if m.DescriptionFn != nil {
		if s, ok := snapshot.(*xs.MachineSnapshot[C]); ok {
			return m.DescriptionFn(s), false
		}
	}
	return m.Description, true
}

// TestStateResult mirrors TestStateResult (also used for the `event` field
// of TestStepResult).
type TestStateResult struct {
	Error error
}

// TestStepResult mirrors TestStepResult.
type TestStepResult[S xs.Snapshot] struct {
	Step  Step[S]
	State TestStateResult
	Event TestStateResult
}

// TestPathResult mirrors TestPathResult.
type TestPathResult[S xs.Snapshot] struct {
	Steps []TestStepResult[S]
	State TestStateResult
}

// EventExecutor mirrors EventExecutor: executes the effect that triggers
// step.Event in the system under test.
type EventExecutor[S xs.Snapshot] func(step Step[S]) error

// TestParam mirrors TestParam. JS iterates `states` in object key order; Go
// iterates state keys in sorted order.
type TestParam[S xs.Snapshot] struct {
	States map[string]func(state S) error
	Events map[string]EventExecutor[S]
}

// TestPath mirrors TestPath.
type TestPath[S xs.Snapshot] struct {
	StatePath[S]
	Description string
	// Test tests and executes each step in Steps sequentially, then tests
	// the postcondition that State is reached.
	Test func(params TestParam[S]) (TestPathResult[S], error)
}

// TestModelLogger mirrors TestModelOptions.logger.
type TestModelLogger struct {
	Log   func(msg string)
	Error func(msg string)
}

// TestModelOptions mirrors TestModelOptions.
type TestModelOptions[S xs.Snapshot] struct {
	TraversalOptions[S]
	StateMatcher        func(state S, stateKey string) bool
	Logger              TestModelLogger
	SerializeTransition func(state S, event xs.Event, prevState S) string
}

// PathGenerator mirrors PathGenerator.
type PathGenerator[S xs.Snapshot] func(logic xs.TypedActorLogic[S], options TraversalOptions[S]) []StatePath[S]

// GetPathOptions mirrors the options of TestModel.getPaths & co.
type GetPathOptions[S xs.Snapshot] struct {
	TraversalOptions[S]
	// AllowDuplicatePaths keeps paths contained by longer paths
	// (default false: paths are deduplicated).
	AllowDuplicatePaths bool
}
