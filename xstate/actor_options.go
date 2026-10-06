package xstate

// ActorOption configures CreateActor.
type ActorOption func(*actorOptions)

type actorOptions struct {
	id               string
	systemID         string
	input            any
	snapshot         any
	inspect          func(InspectionEvent)
	inspectObserver  *Observer[InspectionEvent]
	clock            Clock
	logger           func(args ...any)
	parent           ActorRef
	onUnhandledError func(err any)
	onWarn           func(args ...any)
	src              any
	syncSnapshot     bool
}

func WithID(id string) ActorOption { return func(o *actorOptions) { o.id = id } }

func WithSystemID(systemID string) ActorOption {
	return func(o *actorOptions) { o.systemID = systemID }
}

func WithInput(input any) ActorOption { return func(o *actorOptions) { o.input = input } }

func WithSnapshot(persisted any) ActorOption { return func(o *actorOptions) { o.snapshot = persisted } }

func WithInspect(fn func(InspectionEvent)) ActorOption {
	return func(o *actorOptions) { o.inspect = fn }
}

func WithClock(clock Clock) ActorOption { return func(o *actorOptions) { o.clock = clock } }

func WithLogger(fn func(args ...any)) ActorOption { return func(o *actorOptions) { o.logger = fn } }

func WithParent(parent ActorRef) ActorOption { return func(o *actorOptions) { o.parent = parent } }

// WithUnhandledErrorHandler receives errors JS would report via
// reportUnhandledError (errors with no error observer). Default: log.
func WithUnhandledErrorHandler(fn func(err any)) ActorOption {
	return func(o *actorOptions) { o.onUnhandledError = fn }
}

// WithWarnHandler receives warnings JS writes with console.warn (e.g.
// `Event "PING" was sent to stopped actor "myChild (x:1)". ...`). It mirrors
// `vi.spyOn(console, 'warn')` in the JS tests; it applies to the whole actor
// system of the root actor.
func WithWarnHandler(fn func(args ...any)) ActorOption {
	return func(o *actorOptions) { o.onWarn = fn }
}

// WithInspectObserver mirrors createActor(logic, { inspect: { next } }), the
// observer-object form of the `inspect` option.
func WithInspectObserver(observer Observer[InspectionEvent]) ActorOption {
	return func(o *actorOptions) { o.inspectObserver = &observer }
}
