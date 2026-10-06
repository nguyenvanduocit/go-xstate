package chessmatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const validResponse = `{"answers":{"move":{"type":"choice","choice":"e2e4","confidence":0.75}},"usage":{"cost":0.0001}}`

func TestDecisionsRequest(t *testing.T) {
	requests := make(chan decisionRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-key" || r.URL.Path != "/api/alpha/decisions" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var payload decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		requests <- payload
		fmt.Fprint(w, validResponse)
	}))
	defer server.Close()
	client, err := NewOpenRouter("test-key", time.Second)
	require.NoError(t, err)
	client.endpoint = server.URL + "/api/alpha/decisions"
	turn := Turn{Model: Clef, Side: "white", FEN: "test-fen", Board: "test-board",
		LegalMoves: []MoveOption{{UCI: "e2e4", SAN: "e4"}, {UCI: "d2d4", SAN: "d4"}}}
	decision, err := client.Choose(context.Background(), turn)
	require.NoError(t, err)
	require.Equal(t, "e2e4", decision.Move)
	require.Equal(t, .75, decision.Confidence)
	require.Equal(t, .0001, decision.CostUSD)
	require.Positive(t, decision.Latency)
	payload := <-requests
	require.Equal(t, Clef, payload.Model)
	require.Equal(t, turn.FEN, payload.State.FEN)
	require.Equal(t, turn.Board, payload.State.Board)
	require.Equal(t, "choice", payload.Questions["move"].Type)
	require.Len(t, payload.Questions["move"].Criteria, 2)
	require.Contains(t, payload.Questions["move"].Criteria, "d2d4")
}

func TestRejectsUnusableResponses(t *testing.T) {
	cases := []struct {
		name string
		code int
		body string
	}{
		{"unauthorized", 401, "test-key must never appear in the error"},
		{"rate limit", 429, "retry later"},
		{"not JSON", 200, "bad JSON"},
		{"missing answer", 200, `{}`},
		{"wrong type", 200, strings.Replace(validResponse, `"type":"choice"`, `"type":"score"`, 1)},
		{"illegal choice", 200, strings.Replace(validResponse, "e2e4", "e2e5", 1)},
		{"missing confidence", 200, strings.Replace(validResponse, `,"confidence":0.75`, "", 1)},
		{"out of range confidence", 200, strings.Replace(validResponse, "0.75", "1.5", 1)},
		{"missing cost", 200, strings.Replace(validResponse, `"cost":0.0001`, `"tokens":10`, 1)},
		{"negative cost", 200, strings.Replace(validResponse, "0.0001", "-1", 1)},
		{"oversized", 200, strings.Repeat("x", (1<<20)+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			client, err := NewOpenRouter("test-key", time.Second)
			require.NoError(t, err)
			client.endpoint = server.URL
			_, err = client.Choose(context.Background(), Turn{Model: Jev, LegalMoves: []MoveOption{{UCI: "e2e4"}}})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "test-key")
		})
	}
}

func TestHTTPTimeoutAndCancellation(t *testing.T) {
	for _, useCancel := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Consume the body so the server can detect the client's disconnect.
			var body any
			_ = json.NewDecoder(r.Body).Decode(&body)
			<-r.Context().Done()
		}))
		client, err := NewOpenRouter("test-key", 30*time.Millisecond)
		require.NoError(t, err)
		client.endpoint = server.URL
		ctx, cancel := context.WithCancel(context.Background())
		if useCancel {
			cancel()
		}
		_, err = client.Choose(ctx, Turn{Model: Clef, LegalMoves: []MoveOption{{UCI: "e2e4"}}})
		cancel()
		require.Error(t, err)
		server.Close()
	}
}

func TestDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			t.Error("client followed a redirect with credentials")
		}
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer server.Close()
	client, err := NewOpenRouter("test-key", time.Second)
	require.NoError(t, err)
	client.endpoint = server.URL
	_, err = client.Choose(context.Background(), Turn{LegalMoves: []MoveOption{{UCI: "e2e4"}}})
	require.ErrorContains(t, err, "HTTP 302")
}
