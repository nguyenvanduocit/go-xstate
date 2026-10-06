// Package chessmatch runs a chess game between two decision models.
package chessmatch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	chess "github.com/corentings/chess/v2"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

const (
	Clef = "cloudflare/clef"
	Jev  = "typesafe/jev-1.13"
)

type Config struct {
	WhiteModel string
	BlackModel string
	MaxPlies   int // Half-moves; zero defaults to 1000.
	FEN        string
}

type MoveOption struct {
	UCI string `json:"uci"`
	SAN string `json:"san"`
}

type Turn struct {
	Model       string
	FEN         string
	Board       string
	Side        string
	LegalMoves  []MoveOption
	RecentMoves []string
}

type Decision struct {
	Move       string
	Confidence float64
	CostUSD    float64
	Latency    time.Duration
}

type ChooseMove func(context.Context, Turn) (Decision, error)

type Ply struct {
	Number     int     `json:"number"`
	Side       string  `json:"side"`
	Model      string  `json:"model"`
	UCI        string  `json:"uci"`
	SAN        string  `json:"san"`
	Confidence float64 `json:"confidence"`
	CostUSD    float64 `json:"cost_usd"`
	LatencyMS  int64   `json:"latency_ms"`
}

// Match contains values only; no mutable chess engine or API credentials enter snapshots.
type Match struct {
	InitialFEN string       `json:"initial_fen"`
	FEN        string       `json:"fen"`
	Board      string       `json:"board"`
	Side       string       `json:"side"`
	LegalMoves []MoveOption `json:"legal_moves"`
	Moves      []Ply        `json:"moves"`
	Result     string       `json:"result"`
	Reason     string       `json:"reason"`
	Error      string       `json:"error,omitempty"`
	PGN        string       `json:"pgn"`
	CostUSD    float64      `json:"cost_usd"`
	pending    Decision
}

func NewMachine(cfg Config, choose ChooseMove) (*xs.StateMachine[Match], error) {
	if choose == nil {
		return nil, errors.New("a move chooser is required")
	}
	if cfg.MaxPlies < 0 {
		return nil, errors.New("max plies must be positive")
	}
	if cfg.MaxPlies == 0 {
		cfg.MaxPlies = 1000
	}
	if cfg.WhiteModel == "" {
		cfg.WhiteModel = Clef
	}
	if cfg.BlackModel == "" {
		cfg.BlackModel = Jev
	}
	game := chess.NewGame()
	if cfg.FEN != "" {
		option, err := chess.FEN(cfg.FEN)
		if err != nil {
			return nil, fmt.Errorf("starting FEN: %w", err)
		}
		game = chess.NewGame(option)
	}
	game.AddTagPair("White", cfg.WhiteModel)
	game.AddTagPair("Black", cfg.BlackModel)
	if cfg.FEN != "" {
		game.AddTagPair("SetUp", "1")
		game.AddTagPair("FEN", game.FEN())
	}
	initial := describe(Match{InitialFEN: game.FEN()}, game)
	model := func(side string) string {
		if side == "white" {
			return cfg.WhiteModel
		}
		return cfg.BlackModel
	}
	decide, err := xs.NewTask(choose)
	if err != nil {
		return nil, err
	}
	invocation, err := xs.InvokeTask(decide, xs.Invocation[Match, Turn, Decision]{
		ID: "choose", DoneTarget: "applying", ErrorTarget: "failed",
		Input: func(c Match) Turn {
			recent := make([]string, 0, 8)
			for _, move := range c.Moves[max(0, len(c.Moves)-8):] {
				recent = append(recent, move.SAN)
			}
			return Turn{Model: model(c.Side), FEN: c.FEN, Board: c.Board, Side: c.Side,
				LegalMoves: slices.Clone(c.LegalMoves), RecentMoves: recent}
		},
		Done:   func(c Match, decision Decision) Match { c.pending = decision; return c },
		Failed: func(c Match, err error) Match { c.Error = err.Error(); return c },
	})
	if err != nil {
		return nil, err
	}
	return xs.Compile(xs.MachineConfig[Match]{
		ID: "chess-match", Initial: "setup",
		On: map[string]xs.Transitions{"cancel": {{Target: ".cancelled"}}},
		States: xs.States{
			{Key: "setup", Entry: xs.Actions{xs.Assign(func(xs.AssignArgs[Match]) Match {
				return initial
			})}, Always: xs.Transitions{{Target: "checking"}}},
			{Key: "checking", Always: xs.Transitions{
				{Target: "failed", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return a.Context.Error != "" })},
				{Target: "finished", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return a.Context.Result != "*" })},
				{Target: "limited", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return len(a.Context.Moves) >= cfg.MaxPlies })},
				{Target: "thinking"},
			}},
			{Key: "thinking", Invoke: []xs.InvokeConfig{invocation}},
			{Key: "applying", Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[Match]) Match {
				return apply(a.Context, cfg, model(a.Context.Side))
			})}, Always: xs.Transitions{{Target: "checking"}}},
			{Key: "finished", Type: xs.Final},
			{Key: "limited", Type: xs.Final, Entry: reason("ply limit")},
			{Key: "failed", Type: xs.Final, Entry: reason("error")},
			{Key: "cancelled", Type: xs.Final, Entry: reason("cancelled")},
		},
	})
}

func reason(value string) xs.Actions {
	return xs.Actions{xs.Assign(func(a xs.AssignArgs[Match]) Match {
		c := a.Context
		c.Reason = value
		return c
	})}
}

func describe(c Match, game *chess.Game) Match {
	game.AddTagPair("Result", game.Outcome().String())
	c.FEN, c.Board, c.Result, c.PGN = game.FEN(), game.Position().Board().Draw(), game.Outcome().String(), game.String()
	c.Side = "white"
	if game.Position().Turn() == chess.Black {
		c.Side = "black"
	}
	c.LegalMoves = nil
	if game.Outcome() != chess.NoOutcome {
		c.Reason = game.Method().String()
		return c
	}
	for _, move := range game.ValidMoves() {
		c.LegalMoves = append(c.LegalMoves, MoveOption{
			UCI: chess.UCINotation{}.Encode(game.Position(), &move),
			SAN: chess.AlgebraicNotation{}.Encode(game.Position(), &move),
		})
	}
	sort.Slice(c.LegalMoves, func(i, j int) bool { return c.LegalMoves[i].UCI < c.LegalMoves[j].UCI })
	return c
}

func apply(c Match, cfg Config, model string) Match {
	decision := c.pending
	c.pending = Decision{}
	index := slices.IndexFunc(c.LegalMoves, func(m MoveOption) bool { return m.UCI == decision.Move })
	if index < 0 {
		c.Error = "model selected a move outside the legal choices"
		return c
	}
	// Replay preserves repetition history without sharing a mutable Game across snapshots.
	option, err := chess.FEN(c.InitialFEN)
	if err != nil {
		c.Error = err.Error()
		return c
	}
	game := chess.NewGame(option)
	game.AddTagPair("White", cfg.WhiteModel)
	game.AddTagPair("Black", cfg.BlackModel)
	if cfg.FEN != "" {
		game.AddTagPair("SetUp", "1")
		game.AddTagPair("FEN", c.InitialFEN)
	}
	for _, ply := range c.Moves {
		if err := game.PushNotationMove(ply.UCI, chess.UCINotation{}, nil); err != nil {
			c.Error = fmt.Sprintf("replay: %v", err)
			return c
		}
	}
	if err := game.PushNotationMove(decision.Move, chess.UCINotation{}, nil); err != nil {
		c.Error = fmt.Sprintf("apply move: %v", err)
		return c
	}
	c.Moves = append(slices.Clone(c.Moves), Ply{Number: len(c.Moves) + 1, Side: c.Side, Model: model,
		UCI: decision.Move, SAN: c.LegalMoves[index].SAN, Confidence: decision.Confidence,
		CostUSD: decision.CostUSD, LatencyMS: decision.Latency.Milliseconds()})
	c.CostUSD += decision.CostUSD
	return describe(c, game)
}

// Play stops outstanding model work when ctx is cancelled. onMove runs synchronously.
func Play(ctx context.Context, cfg Config, choose ChooseMove, onMove func(Ply)) (Match, error) {
	if err := ctx.Err(); err != nil {
		return Match{}, err
	}
	machine, err := NewMachine(cfg, choose)
	if err != nil {
		return Match{}, err
	}
	actor := xs.CreateActor(machine)
	defer actor.Stop()
	seen := 0
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Match]]{
		Next: func(s *xs.MachineSnapshot[Match]) {
			for seen < len(s.Context.Moves) {
				move := s.Context.Moves[seen]
				seen++
				if onMove != nil {
					onMove(move)
				}
			}
		},
	})
	actor.Start()
	_, err = xs.Await(ctx, actor, func(s *xs.MachineSnapshot[Match]) bool { return s.Status == xs.StatusDone })
	if err != nil && ctx.Err() != nil {
		actor.Send(xs.Ev("cancel"))
		err = ctx.Err()
	}

	result := actor.GetSnapshot().Context
	if err == nil && result.Error != "" {
		err = errors.New(result.Error)
	}
	return result, err
}
