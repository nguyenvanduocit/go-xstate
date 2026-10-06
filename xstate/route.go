package xstate

// formatRouteTransitions mirrors stateUtils.ts formatRouteTransitions.
func formatRouteTransitions(root *StateNode) {
	var routes []*TransitionDefinition
	var collect func(n *StateNode)
	collect = func(n *StateNode) {
		for _, sn := range n.ChildStates() {
			if sn.Config.Route != nil && sn.Config.ID != "" {
				routeID := sn.Config.ID
				routeMatches := &inlineGuard{fn: func(a guardArgs, _ any) bool {
					to := eventField(a.event, "to")
					s, _ := to.(string)
					return s == "#"+routeID
				}}
				cfg := *sn.Config.Route
				if cfg.Guard != nil {
					cfg.Guard = And(routeMatches, cfg.Guard)
				} else {
					cfg.Guard = routeMatches
				}
				cfg.Target = "#" + routeID
				cfg.Targets = nil
				routes = append(routes, formatTransition(root, "xstate.route", cfg))
			}
			collect(sn)
		}
	}
	collect(root)
	if len(routes) > 0 {
		root.transitions.set("xstate.route", routes)
	}
}

// eventField reads a payload field of an event (`event[key]`).
func eventField(e Event, key string) any {
	switch ev := e.(type) {
	case E:
		return ev[key]
	}
	return nil
}
