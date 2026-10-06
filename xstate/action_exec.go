package xstate

// ---- action resolution ----

type resolveExtra struct {
	internalQueue    *[]Event
	deferredActorIDs []string
	hasDeferred      bool
}

type retryEntry struct {
	action *builtinAction
	params any
}

func resolveParams(params any, context any, event Event, scope *ActorScope) any {
	if e, ok := params.(*exprFunc); ok {
		a := exprArgs{context: context, event: event}
		if scope != nil {
			a.self = scope.Self
			a.system = scope.System
		}
		return e.fn(a)
	}
	return params
}

func resolveAndExecuteActionsWithContext(current anyMachineSnapshot, event Event, scope *ActorScope, actions Actions, extra *resolveExtra, retries *[]retryEntry) anyMachineSnapshot {
	m := current.machineAny()
	inter := current
	for _, action := range actions {
		if action == nil {
			continue
		}
		var resolved Action
		var params any
		var typ string
		isInline := false
		switch a := action.(type) {
		case *builtinAction:
			resolved, isInline = a, true
		case *inlineAction:
			resolved, isInline = a, true
		case ActionRef:
			typ = a.Type
			resolved = m.impl().Actions[a.Type]
			params = resolveParams(a.Params, inter.contextAny(), event, scope)
		case *ActionRef:
			typ = a.Type
			resolved = m.impl().Actions[a.Type]
			params = resolveParams(a.Params, inter.contextAny(), event, scope)
		default:
			typ = a.ActionType()
			resolved = m.impl().Actions[typ]
		}
		args := actionArgs{context: inter.contextAny(), event: event, self: scope.Self, system: scope.System}
		b, isBuiltin := resolved.(*builtinAction)
		if !isBuiltin {
			if isInline {
				typ = "(anonymous)"
			}
			var exec func()
			if inl, ok := resolved.(*inlineAction); ok {
				p := params
				exec = func() { inl.fn(args, p) }
			}
			scope.actionExecutor(ExecutableAction{Type: typ, Info: args.public(), Params: params, Exec: exec})
			continue
		}
		next, p, more := b.resolve(scope, inter, args, params, extra)
		inter = next
		if b.retryResolve != nil && retries != nil {
			*retries = append(*retries, retryEntry{b, p})
		}
		if b.execute != nil {
			exec := b.execute
			scope.actionExecutor(ExecutableAction{Type: b.typ, Info: args.public(), Params: p, Exec: func() { exec(scope, p) }})
		}
		if more != nil {
			inter = resolveAndExecuteActionsWithContext(inter, event, scope, more, extra, retries)
		}
	}
	return inter
}

func resolveActionsAndContext(current anyMachineSnapshot, event Event, scope *ActorScope, actions Actions, internalQueue *[]Event, deferredActorIDs []string, hasDeferred bool) anyMachineSnapshot {
	var retries *[]retryEntry
	if hasDeferred {
		retries = &[]retryEntry{}
	}
	next := resolveAndExecuteActionsWithContext(current, event, scope, actions, &resolveExtra{
		internalQueue:    internalQueue,
		deferredActorIDs: deferredActorIDs,
		hasDeferred:      hasDeferred,
	}, retries)
	if retries != nil {
		for _, r := range *retries {
			r.action.retryResolve(scope, next, r.params)
		}
	}
	return next
}
