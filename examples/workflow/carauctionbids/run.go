package carauctionbids

import (
	"fmt"
	"io"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Run mirrors main.ts: it starts the machine on the real clock, sends two bids one second apart
// and prints every received event and every snapshot context until BiddingDelay (3 s) ends the
// auction. It returns when the workflow has completed.
func Run(w io.Writer) {
	completed := make(chan struct{})
	actor := xs.CreateActor(Machine(), xs.WithInspect(func(e xs.InspectionEvent) {
		if e.Type == xs.InspectEvent {
			fmt.Fprintln(w, "Received event", inspect(e.Event))
		}
	}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) {
			fmt.Fprintln(w, inspect(s.Context))
		},
		Complete: func() {
			fmt.Fprintln(w, "workflow completed", inspect(actor.GetSnapshot().Output))
			close(completed)
		},
	})
	actor.Start()

	time.Sleep(1000 * time.Millisecond)
	actor.Send(CarBidEvent(Bid{CarID: "car123", Amount: 3000, Bidder: Bidder{ID: "xyz", FirstName: "John", LastName: "Wayne"}}))

	time.Sleep(1000 * time.Millisecond)
	actor.Send(CarBidEvent(Bid{CarID: "car123", Amount: 4000, Bidder: Bidder{ID: "abc", FirstName: "Jane", LastName: "Doe"}}))

	<-completed
}
