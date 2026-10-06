// Package carauctionbids ports references/xstate/examples/workflow-car-auction-bids/main.ts
// (https://github.com/serverlessworkflow/specification/tree/main/examples#handle-car-auction-bids-example).
package carauctionbids

import (
	"encoding/json"
	"errors"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// BiddingDelay is the `BiddingDelay` delay of the JS machine (3000 ms).
const BiddingDelay = 3000 * time.Millisecond

// Bidder mirrors `Bid.bidder`.
type Bidder struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Bid mirrors the `Bid` interface.
type Bid struct {
	CarID  string  `json:"carid"`
	Amount float64 `json:"amount"`
	Bidder Bidder  `json:"bidder"`
}

// Context is the machine's extended state.
type Context struct {
	Bids []Bid `json:"bids"`
}

// Output mirrors the object returned by BiddingEnded's `output`.
type Output struct {
	WinningBid Bid `json:"winningBid"`
}

// CarBidEvent builds `{ type: 'CarBidEvent', bid }`.
func CarBidEvent(bid Bid) xs.E {
	return xs.E{"type": "CarBidEvent", "bid": bid}
}

// WinningBid mirrors `bids.reduce((prev, current) => prev.amount > current.amount ? prev : current)`:
// the highest amount wins and a tie goes to the later bid. Like Array.prototype.reduce without an
// initial value it throws on an empty list (JavaScriptCore's message, as recorded by the Bun trace).
func WinningBid(bids []Bid) Bid {
	if len(bids) == 0 {
		panic(errors.New("reduce of empty array with no initial value"))
	}
	prev := bids[0]
	for _, current := range bids[1:] {
		if !(prev.Amount > current.Amount) {
			prev = current
		}
	}
	return prev
}

// bidOf reads `event.bid`. The payload is a Bid when built by CarBidEvent and a decoded JSON
// object when replayed from a golden file; a JSON round trip handles both.
func bidOf(event xs.Event) Bid {
	raw, err := json.Marshal(event.(xs.E)["bid"])
	if err != nil {
		panic(err)
	}
	var bid Bid
	if err := json.Unmarshal(raw, &bid); err != nil {
		panic(err)
	}
	return bid
}

// Machine mirrors `workflow`.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:          "handleCarAuctionBid",
		Description: "Store a single bid whole the car auction is active",
		Initial:     "StoreCarAuctionBid",
		Context:     Context{Bids: []Bid{}},
		States: xs.States{
			{
				Key: "StoreCarAuctionBid",
				On: map[string]xs.Transitions{
					"CarBidEvent": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[Context]) Context {
							bids := append(append([]Bid{}, a.Context.Bids...), bidOf(a.Event))
							return Context{Bids: bids}
						}),
					}}},
				},
				After: map[string]xs.Transitions{
					"BiddingDelay": {{Target: "BiddingEnded"}},
				},
			},
			{
				Key:  "BiddingEnded",
				Type: xs.Final,
				Output: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
					// highest bid
					return Output{WinningBid: WinningBid(a.Context.Bids)}
				}),
			},
		},
	}, xs.Implementations{
		Delays: map[string]any{"BiddingDelay": BiddingDelay},
	})
}
