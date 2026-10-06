package carauctionbids_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	auction "github.com/nguyenvanduocit/go-xstate/examples/workflow/carauctionbids"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func replay(t *testing.T, golden string) {
	tracetest.Run(t, golden, func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[auction.Context]] {
		return xs.CreateActor(auction.Machine(), xs.WithClock(clock))
	})
}

// TestTrace replays testdata/workflow-car-auction-bids.golden.json, recorded from the JS machine by
// scripts/trace/workflow-car-auction-bids/workflow-car-auction-bids.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-auction-bids/main.ts#L14
// JS trace: scripts/trace/workflow-car-auction-bids/workflow-car-auction-bids.ts.
func TestTrace(t *testing.T) {
	replay(t, "testdata/workflow-car-auction-bids.golden.json")
}

// TestTraceNoBids replays testdata/no-bids.golden.json (scripts/trace/workflow-car-auction-bids/no-bids.ts):
// the final state's output throws on an empty bid list, which puts the actor in status "error".
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-auction-bids/main.ts#L14
// JS trace: scripts/trace/workflow-car-auction-bids/no-bids.ts.
func TestTraceNoBids(t *testing.T) {
	replay(t, "testdata/no-bids.golden.json")
}

// TestWinningBid compares WinningBid with the JS output function of BiddingEnded, recorded by
// scripts/trace/workflow-car-auction-bids/winning-bid.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-auction-bids/main.ts#L14
// JS trace: scripts/trace/workflow-car-auction-bids/winning-bid.ts.
func TestWinningBid(t *testing.T) {
	raw, err := os.ReadFile("testdata/winning-bid.golden.json")
	require.NoError(t, err)
	var g struct {
		Cases []struct {
			Name   string          `json:"name"`
			Bids   []auction.Bid   `json:"bids"`
			Output *auction.Output `json:"output"`
			Error  string          `json:"error"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Cases)
	for _, c := range g.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Error != "" {
				assert.PanicsWithError(t, c.Error, func() { auction.WinningBid(c.Bids) })
				return
			}
			require.NotNil(t, c.Output)
			assert.Equal(t, c.Output.WinningBid, auction.WinningBid(c.Bids))
		})
	}
}

// TestRun compares Run's output with testdata/workflow-car-auction-bids.stdout.txt, the text printed by
// the JS entry main.ts (recorded by workflow-car-auction-bids.stdout.ts). Takes about 3 s of real time.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-auction-bids/main.ts#L62
// JS trace: scripts/trace/workflow-car-auction-bids/workflow-car-auction-bids.stdout.ts.
func TestRun(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-car-auction-bids.stdout.txt")
	require.NoError(t, err)
	var got bytes.Buffer
	auction.Run(&got)
	assert.Equal(t, string(want), got.String())
}
