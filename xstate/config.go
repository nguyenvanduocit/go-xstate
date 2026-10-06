package xstate

// StateType mirrors StateNode `type`.
type StateType string

const (
	Atomic   StateType = "atomic"
	Compound StateType = "compound"
	Parallel StateType = "parallel"
	Final    StateType = "final"
	History  StateType = "history"
)

// HistoryType mirrors `history: 'shallow' | 'deep'`.
type HistoryType string

const (
	Shallow HistoryType = "shallow"
	Deep    HistoryType = "deep"
)

// TransitionConfig mirrors a single JS transition object.
//
// A JS string shorthand `'b'` becomes `{{Target: "b"}}`.
// A JS array of targets becomes Targets.
type TransitionConfig struct {
	Target      string   // single target ("b", ".child", "#id", "parent.child")
	Targets     []string // multiple targets (parallel regions); used instead of Target
	Guard       Guard
	Actions     Actions
	Reenter     bool
	Meta        any
	Description string
}

// Transitions is an ordered list of candidate transitions for one event.
// A nil Transitions stored under a key mirrors JS `EVENT: undefined`
// (a forbidden transition).
type Transitions []TransitionConfig

// Actions is an ordered list of actions.
type Actions []Action

// Tags is a list of state tags.
type Tags []string

// StateConfig mirrors a JS state node config. Key is the property name the
// state had in the parent's `states` object; States keeps document order.
type StateConfig struct {
	Key         string
	ID          string
	Type        StateType
	History     HistoryType // for Type == History; "" means shallow
	Initial     string
	States      States
	Invoke      []InvokeConfig
	On          map[string]Transitions
	Entry       Actions
	Exit        Actions
	OnDone      Transitions
	After       map[string]Transitions // key: milliseconds ("1000") or a delay name
	Always      Transitions
	Meta        any
	Output      any // static value or Expr
	Tags        Tags
	Description string
	Target      string // default target of a history state
	Route       *TransitionConfig

	initial initialTransitionConfig
}

// initialTransitionConfig holds the parts of a JS initial-transition object
// (`initial: { target, actions, meta, description }`) beyond the target.
type initialTransitionConfig struct {
	isObject    bool
	actions     Actions
	meta        any
	description string
}

// States is the ordered list of child state configs.
type States []StateConfig

// InvokeConfig mirrors a JS invoke config. Exactly one of Src or Logic is set.
type InvokeConfig struct {
	ID         string
	SystemID   string
	Src        string     // name of an actor in Implementations.Actors
	Logic      ActorLogic // inline actor logic
	Input      any        // static value or Expr
	OnDone     Transitions
	OnError    Transitions
	OnSnapshot Transitions
}

// MachineConfig mirrors the root config of createMachine.
type MachineConfig[C any] struct {
	ID          string
	Version     string
	Context     C
	ContextFn   func(ContextArgs) C // used instead of Context when set
	Type        StateType
	Initial     string
	States      States
	Invoke      []InvokeConfig
	On          map[string]Transitions
	Entry       Actions
	Exit        Actions
	OnDone      Transitions
	After       map[string]Transitions
	Always      Transitions
	Meta        any
	Output      any // static value or Expr
	Tags        Tags
	Description string

	initial initialTransitionConfig
	schemas any
	options *MachineOptions
}

// ContextArgs are passed to MachineConfig.ContextFn.
type ContextArgs struct {
	Input any
	Self  ActorRef
	Spawn Spawner
}

// Implementations mirrors the second argument of createMachine / setup().
type Implementations struct {
	Actions map[string]Action
	Guards  map[string]Guard
	Actors  map[string]ActorLogic
	Delays  map[string]any // time.Duration or Expr returning time.Duration
}

// Spawner spawns a child actor from logic or a named actor source.
type Spawner func(src any, opts ...SpawnOptions) ActorRef

// SpawnOptions mirrors spawn / spawnChild options.
type SpawnOptions struct {
	ID           any // string or Expr returning string
	SystemID     string
	Input        any // static value or Expr
	SyncSnapshot bool
}

// SendOptions mirrors sendTo / raise options.
type SendOptions struct {
	ID    any // string or Expr returning string
	Delay any // time.Duration, delay name (string) or Expr returning time.Duration
}

// WithStateInitialActions mirrors a JS state node whose `initial` is an
// initial-transition object: `initial: { target: cfg.Initial, actions }`.
func WithStateInitialActions(cfg StateConfig, actions ...Action) StateConfig {
	cfg.initial.isObject = true
	cfg.initial.actions = append(Actions{}, actions...)
	return cfg
}

// WithMachineInitialActions mirrors a JS root config whose `initial` is an
// initial-transition object: `initial: { target: cfg.Initial, actions }`.
func WithMachineInitialActions[C any](cfg MachineConfig[C], actions ...Action) MachineConfig[C] {
	cfg.initial.isObject = true
	cfg.initial.actions = append(Actions{}, actions...)
	return cfg
}

// WithMachineInitialMeta mirrors a JS root config whose `initial` is an
// initial-transition object carrying metadata: `initial: { target, meta }`.
func WithMachineInitialMeta[C any](cfg MachineConfig[C], meta any) MachineConfig[C] {
	cfg.initial.isObject = true
	cfg.initial.meta = meta
	return cfg
}

// MachineOptions mirrors MachineOptions, the `options` property of the
// createMachine config (`createMachine({ ..., options: { maxIterations } })`).
type MachineOptions struct {
	// MaxIterations mirrors `maxIterations`: the maximum number of microsteps
	// processed before an "Infinite loop detected" error. 0 means Infinity
	// (the JS default).
	MaxIterations int
}

// WithOptions mirrors passing `options` in the createMachine config; it
// returns a machine whose runtime options are opts.
func (m *StateMachine[C]) WithOptions(opts MachineOptions) *StateMachine[C] {
	cfg := m.Config
	cfg.options = &opts
	return newStateMachine(cfg, m.Implementations)
}

// WithSchemas mirrors the `schemas` key of `setup({ schemas, ... })`.
func (s *Setup[C]) WithSchemas(schemas any) *Setup[C] {
	return &Setup[C]{Implementations: s.Implementations, schemas: schemas}
}

// Schemas mirrors `machine.schemas`.
func (m *StateMachine[C]) Schemas() any { return m.Config.schemas }
