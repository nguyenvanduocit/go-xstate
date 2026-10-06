package xstate

// ---- setup ----

// Setup mirrors setup({ actions, guards, actors, delays }).
type Setup[C any] struct {
	Implementations Implementations

	schemas any
}

// NewSetup mirrors setup(implementations).
func NewSetup[C any](impl Implementations) *Setup[C] { return &Setup[C]{Implementations: impl} }

// CreateMachine mirrors setup(...).createMachine(config).
func (s *Setup[C]) CreateMachine(config MachineConfig[C]) *StateMachine[C] {
	config.schemas = s.schemas
	return CreateMachine(config, s.Implementations)
}

// Extend mirrors setup(...).extend(implementations).
func (s *Setup[C]) Extend(impl Implementations) *Setup[C] {
	return &Setup[C]{
		Implementations: Implementations{
			Actors:  s.Implementations.Actors,
			Actions: mergeMaps(s.Implementations.Actions, impl.Actions),
			Guards:  mergeMaps(s.Implementations.Guards, impl.Guards),
			Delays:  mergeMaps(s.Implementations.Delays, impl.Delays),
		},
		schemas: s.schemas,
	}
}

// CreateAction mirrors setup(...).createAction(fn).
func (s *Setup[C]) CreateAction(fn func(ActionArgs[C])) Action { return ActionFunc(fn) }

// CreateStateConfig mirrors setup(...).createStateConfig(config).
func (s *Setup[C]) CreateStateConfig(config StateConfig) StateConfig { return config }
