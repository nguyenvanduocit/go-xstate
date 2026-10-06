// Package badukmatch runs Go (Baduk/Weiqi) between decision models.
package badukmatch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

const (
	Clef = "cloudflare/clef"
	Jev  = "typesafe/jev-1.13"
)

type Config struct {
	Size       int     // Zero defaults to 9; supported sizes are 2 through 19.
	Komi       float64 // Added to White's area; zero means no komi.
	BlackModel string
	WhiteModel string
	MaxMoves   int // Zero defaults to 1000, including passes.
}

type Turn struct {
	Model             string
	Size              int
	Komi              float64
	Board             string
	Side              string
	MoveNumber        int
	ConsecutivePasses int
	LegalMoves        []string
	RecentMoves       []string
}

type Decision struct {
	Move       string
	Confidence float64
	CostUSD    float64
	Latency    time.Duration
}

type ChooseMove func(context.Context, Turn) (Decision, error)

type Move struct {
	Number     int     `json:"number"`
	Side       string  `json:"side"`
	Model      string  `json:"model"`
	Point      string  `json:"point"`
	Confidence float64 `json:"confidence"`
	CostUSD    float64 `json:"cost_usd"`
	LatencyMS  int64   `json:"latency_ms"`
}

type Match struct {
	Size              int      `json:"size"`
	Komi              float64  `json:"komi"`
	Board             string   `json:"board"`
	Cells             string   `json:"cells"`
	Side              string   `json:"side"`
	ConsecutivePasses int      `json:"consecutive_passes"`
	LegalMoves        []string `json:"legal_moves"`
	Moves             []Move   `json:"moves"`
	Result            string   `json:"result"` // "*" until scored, then B+margin, W+margin, or 0.
	Reason            string   `json:"reason"`
	Error             string   `json:"error,omitempty"`
	Score             Score    `json:"score"`
	Scored            bool     `json:"scored"`
	SGF               string   `json:"sgf"`
	CostUSD           float64  `json:"cost_usd"`
	position          position
	pending           Decision
}

func NewMachine(cfg Config, choose ChooseMove) (*xs.StateMachine[Match], error) {
	if choose == nil {
		return nil, errors.New("a move chooser is required")
	}
	if cfg.Size == 0 {
		cfg.Size = 9
	}
	if cfg.Size < 2 || cfg.Size > 19 {
		return nil, errors.New("board size must be between 2 and 19")
	}
	if math.IsNaN(cfg.Komi) || math.IsInf(cfg.Komi, 0) || math.Mod(cfg.Komi*2, 1) != 0 {
		return nil, errors.New("komi must be a finite multiple of 0.5")
	}
	if cfg.MaxMoves < 0 {
		return nil, errors.New("max moves must be positive")
	}
	if cfg.MaxMoves == 0 {
		cfg.MaxMoves = 1000
	}
	if cfg.BlackModel == "" {
		cfg.BlackModel = Clef
	}
	if cfg.WhiteModel == "" {
		cfg.WhiteModel = Jev
	}
	model := func(side string) string {
		if side == "black" {
			return cfg.BlackModel
		}
		return cfg.WhiteModel
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
				recent = append(recent, move.Side+" "+move.Point)
			}
			return Turn{Model: model(c.Side), Size: c.Size, Komi: c.Komi, Board: c.Board,
				Side: c.Side, MoveNumber: len(c.Moves) + 1, ConsecutivePasses: c.ConsecutivePasses,
				LegalMoves: slices.Clone(c.LegalMoves), RecentMoves: recent}
		},
		Done:   func(c Match, decision Decision) Match { c.pending = decision; return c },
		Failed: func(c Match, err error) Match { c.Error = err.Error(); return c },
	})
	if err != nil {
		return nil, err
	}
	return xs.Compile(xs.MachineConfig[Match]{
		ID: "baduk-match", Initial: "setup",
		On: map[string]xs.Transitions{"cancel": {{Target: ".cancelled"}}},
		States: xs.States{
			{Key: "setup", Entry: xs.Actions{xs.Assign(func(xs.AssignArgs[Match]) Match {
				return describe(Match{Size: cfg.Size, Komi: cfg.Komi, Result: "*", position: newPosition(cfg.Size)}, cfg)
			})}, Always: xs.Transitions{{Target: "checking"}}},
			{Key: "checking", Always: xs.Transitions{
				{Target: "failed", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return a.Context.Error != "" })},
				{Target: "scoring", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return a.Context.ConsecutivePasses == 2 })},
				{Target: "limited", Guard: xs.GuardFunc(func(a xs.GuardArgs[Match]) bool { return len(a.Context.Moves) >= cfg.MaxMoves })},
				{Target: "thinking"},
			}},
			{Key: "thinking", Invoke: []xs.InvokeConfig{invocation}},
			{Key: "applying", Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[Match]) Match {
				c := a.Context
				decision := c.pending
				c.pending = Decision{}
				if !slices.Contains(c.LegalMoves, decision.Move) {
					c.Error = "model selected a move outside the legal choices"
					return c
				}
				next, err := c.position.play(decision.Move)
				if err != nil {
					c.Error = err.Error()
					return c
				}
				c.Moves = append(slices.Clone(c.Moves), Move{Number: len(c.Moves) + 1, Side: c.Side,
					Model: model(c.Side), Point: decision.Move, Confidence: decision.Confidence,
					CostUSD: decision.CostUSD, LatencyMS: decision.Latency.Milliseconds()})
				c.CostUSD += decision.CostUSD
				c.position = next
				return describe(c, cfg)
			})}, Always: xs.Transitions{{Target: "checking"}}},
			{Key: "scoring", Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[Match]) Match {
				c := a.Context
				c.Score, c.Scored = c.position.score(c.Komi), true
				c.Result, c.Reason = c.Score.result(), "two consecutive passes"
				c.SGF = sgf(c, cfg)
				return c
			})}, Always: xs.Transitions{{Target: "finished"}}},
			{Key: "finished", Type: xs.Final},
			{Key: "limited", Type: xs.Final, Entry: reason("move limit")},
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

func describe(c Match, cfg Config) Match {
	c.Board, c.Cells, c.Side = c.position.draw(), c.position.cells, c.position.side()
	c.ConsecutivePasses, c.LegalMoves = c.position.passes, c.position.legalMoves()
	c.SGF = sgf(c, cfg)
	return c
}

func sgf(c Match, cfg Config) string {
	escape := func(s string) string {
		return strings.NewReplacer("\\", "\\\\", "]", "\\]", "\r", " ", "\n", " ", "\t", " ").Replace(s)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "(;GM[1]FF[4]CA[UTF-8]SZ[%d]KM[%s]RU[Tromp-Taylor]PB[%s]PW[%s]",
		c.Size, strconv.FormatFloat(c.Komi, 'f', -1, 64), escape(cfg.BlackModel), escape(cfg.WhiteModel))
	if c.Scored {
		fmt.Fprintf(&out, "RE[%s]", c.Result)
	}
	for _, move := range c.Moves {
		color, coordinate := "B", ""
		if move.Side == "white" {
			color = "W"
		}
		if move.Point != "pass" {
			index, _ := point(move.Point, c.Size)
			coordinate = string([]byte{byte('a' + index%c.Size), byte('a' + index/c.Size)})
		}
		fmt.Fprintf(&out, ";%s[%s]", color, coordinate)
	}
	out.WriteByte(')')
	return out.String()
}

// DemoChoose plays twelve deterministic placements, then passes; it is not an AI opponent.
func DemoChoose(_ context.Context, turn Turn) (Decision, error) {
	placements := len(turn.LegalMoves) - 1 // pass is last
	if turn.MoveNumber > 12 || placements == 0 {
		return Decision{Move: "pass"}, nil
	}
	return Decision{Move: turn.LegalMoves[(turn.MoveNumber*7)%placements]}, nil
}

// Play stops outstanding model work when ctx is cancelled. onMove runs synchronously.
func Play(ctx context.Context, cfg Config, choose ChooseMove, onMove func(Move)) (Match, error) {
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
