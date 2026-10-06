> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: route_1

Source: `references/xstate/packages/core/test/route.test.ts` lines 1-491 (whole file, 490 lines).
Go file: `xstate/route_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| route > should transition directly to a route if route is an empty transition config (L4) | TestRoute_ShouldTransitionDirectlyToRouteIfRouteIsEmptyTransitionConfig | ported | |
| route > should transition directly to a route if guard passes (L36) | TestRoute_ShouldTransitionDirectlyToRouteIfGuardPasses | ported | |
| route > should resolve setup-registered string guards on route transitions (L76) | TestRoute_ShouldResolveSetupRegisteredStringGuardsOnRouteTransitions | ported | `guard: 'isReady'` → `xs.GuardRef{Type: "isReady"}` |
| route > should work with parallel states (L127) | TestRoute_ShouldWorkWithParallelStates | ported | |
| route > route events are strongly typed (L177) | TestRoute_RouteEventsAreStronglyTyped | ported | Runtime part ported (L177-231): machine build, 5 `xstate.route` sends (`#aRoute`, `#childRoute`, invalid `notARoute`/`root`/`blahblah` as plain maps; TS-only compile errors, ignored at runtime). `types.events: never` and `@ts-expect-error` dropped. JS has no `expect`; three value assertions were ADDED (after the two valid routes and after the invalid sends) to make the sends observable. |
| route > route config without id should not generate route events (L233) | TestRoute_RouteConfigWithoutIDShouldNotGenerateRouteEvents | ported | `types.events: never` dropped (type-only) |
| route > machine.root.on should include route events (L264) | TestRoute_MachineRootOnShouldIncludeRouteEvents | ported | `toBeDefined()` → `assert.Contains(machine.Root.On(), "xstate.route")` |
| route > nested state on should include route events for child routes (L286) | TestRoute_NestedStateOnShouldIncludeRouteEventsForChildRoutes | ported | |
| route > parallel state on should include route events (L325) | TestRoute_ParallelStateOnShouldIncludeRouteEvents | ported | |
| route > should route to deeply nested state from anywhere (L361) | TestRoute_ShouldRouteToDeeplyNestedStateFromAnywhere | ported | |
| route > should re-enter when routing to the current state (L392) | TestRoute_ShouldReEnterWhenRoutingToCurrentState | ported | |
| route > should route to self with guard (L418) | TestRoute_ShouldRouteToSelfWithGuard | ported | |
| route > should not route using dot-separated nested id like #id.nested (L449) | TestRoute_ShouldNotRouteUsingDotSeparatedNestedIDLikeIDNested | ported | |

Totals: 13 JS tests; 13 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. `StateConfig.Route *TransitionConfig` (config.go:68) and `StateNode.On()` (machine.go:34) already cover this file.
Route events are sent as `xs.E{"type": "xstate.route", "to": "#id"}` via the file-local helper `route1Ev`.

## Ambiguities for reviewer

- L177-231: "route events are strongly typed" has no `expect` in JS. The Go test ports the machine and all 5 sends; the added assertions (`"aRoute"`, `{"notARoute":"childRoute"}`, unchanged after invalid targets) are not in the JS and assume invalid `to` values are ignored (consistent with L28-33 where `#c` without route is ignored). Reviewer: drop them if undesired.
- L264/L286/L325: `machine.root.on['xstate.route']` → `machine.Root.On()` (StateNode.On returns
  `map[string][]*TransitionDefinition`); `toBeDefined` mapped to key presence.
- `setup({ types: { events: {} as never } })` (L178, L234, L450) mapped to `xs.NewSetup[any](xs.Implementations{})`.
- `entries`/`allowed` counters (L393, L419) are plain locals mutated by synchronous entry actions/guards, as in JS.
