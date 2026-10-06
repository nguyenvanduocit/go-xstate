package xstate

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAwaitCancellationRemovesSubscriptionWithoutStoppingActor(t *testing.T) {
	actor := CreateActor(FromTransition(func(n int, _ Event, _ *ActorScope) int { return n + 1 }, nil)).Start()
	defer actor.Stop()
	cause := errors.New("caller left")
	ctx, cancel := context.WithCancelCause(context.Background())
	entered := make(chan struct{})
	var once sync.Once
	done := make(chan error, 1)
	go func() {
		_, err := Await(ctx, actor, func(*TransitionSnapshot[int]) bool { once.Do(func() { close(entered) }); return false })
		done <- err
	}()
	<-entered
	cancel(cause)
	select {
	case err := <-done:
		if !errors.Is(err, cause) {
			t.Fatalf("lost cause: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waiter did not cancel")
	}
	actor.system.lock()
	for _, o := range actor.observers {
		if !o.removed {
			t.Error("waiter subscription retained")
		}
	}
	actor.system.unlock()
	actor.Send(Ev("inc"))
	if got := actor.GetSnapshot(); got.GetStatus() != StatusActive || got.Context != 1 {
		t.Fatalf("actor stopped by waiter: %+v", got)
	}
}

func TestAwaitAlreadyFinished(t *testing.T) {
	actor := CreateActor(CreateMachine(MachineConfig[int]{Context: 5, Type: Final})).Start()
	defer actor.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshot, err := Await(ctx, actor, func(s *MachineSnapshot[int]) bool { return s.Status == StatusDone })
	if err != nil || snapshot.Context != 5 {
		t.Fatalf("snapshot=%v err=%v", snapshot, err)
	}
}

func TestAwaitInvalidArguments(t *testing.T) {
	actor := CreateActor(CreateMachine(MachineConfig[int]{})).Start()
	defer actor.Stop()
	predicate := func(*MachineSnapshot[int]) bool { return true }
	if _, err := Await(nil, actor, predicate); err == nil {
		t.Error("nil context accepted")
	}
	if _, err := Await(context.Background(), (*Actor[*MachineSnapshot[int]])(nil), predicate); err == nil {
		t.Error("nil actor accepted")
	}
	if _, err := Await(context.Background(), actor, nil); err == nil {
		t.Error("nil predicate accepted")
	}
}

type compilePanicMachine struct{ machineInternal }

func (compilePanicMachine) getStateNodeByID(string) *StateNode { panic("programmer panic") }

func TestConfigResolutionPreservesProgrammerPanic(t *testing.T) {
	defer func() {
		if got := recover(); got != "programmer panic" {
			t.Fatalf("unexpected panic: %v", got)
		}
	}()
	getStateNodeByPath(&StateNode{machine: compilePanicMachine{}}, "#missing")
}
