package xstate

import "fmt"

// createSpawner mirrors spawn.ts createSpawner.
func createSpawner(scope *ActorScope, snap anyMachineSnapshot, event Event, spawned *childSet) Spawner {
	machine := snap.machineAny()
	context := snap.contextAny()
	spawn := func(src any, opts ...SpawnOptions) ActorRef {
		var o SpawnOptions
		if len(opts) > 0 {
			o = opts[0]
		}
		id, _ := evalExpr(o.ID, exprArgs{context: context, event: event, self: scope.Self, system: scope.System}).(string)
		if s, ok := src.(string); ok {
			logic := machine.resolveReferencedActor(s)
			if logic == nil {
				panic(fmt.Errorf("Actor logic '%s' not implemented in machine '%s'", s, machine.MachineID()))
			}
			ref := createChildActor(logic, childOptions{
				id:           id,
				parent:       scope.Self,
				syncSnapshot: o.SyncSnapshot,
				input:        evalExpr(o.Input, exprArgs{context: context, event: event, self: scope.Self, system: scope.System}),
				src:          s,
				systemID:     o.SystemID,
			})
			spawned.set(ref.ID(), ref)
			return ref
		}
		logic, ok := src.(ActorLogic)
		if !ok {
			panic(fmt.Errorf("xstate: cannot spawn %T", src))
		}
		return createChildActor(logic, childOptions{
			id:           id,
			parent:       scope.Self,
			syncSnapshot: o.SyncSnapshot,
			input:        o.Input,
			src:          src,
			systemID:     o.SystemID,
		})
	}
	return func(src any, opts ...SpawnOptions) ActorRef {
		ref := spawn(src, opts...)
		spawned.set(ref.ID(), ref)
		scope.Defer(func() {
			if c := coreOf(ref); c != nil && c.processingStatus == statusStopped {
				return
			}
			startRef(ref)
		})
		return ref
	}
}

func (c *childSet) set(id string, ref ActorRef) {
	if c.m == nil {
		c.m = map[string]ActorRef{}
	}
	if _, ok := c.m[id]; !ok {
		c.order = append(c.order, id)
	}
	c.m[id] = ref
}
