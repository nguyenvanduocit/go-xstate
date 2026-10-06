// Package tictactoe ports references/xstate/examples/tic-tac-toe-react/src/ticTacToeMachine.ts.
package tictactoe

import (
	"encoding/json"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Player is 'x' | 'o'. The empty Player stands for JS `null` (board cell) and
// `undefined` (winner): it marshals to JSON null and is omitted by `omitempty`.
type Player string

const (
	X Player = "x"
	O Player = "o"
)

// MarshalJSON renders the empty Player as null, like an unplayed JS board cell.
func (p Player) MarshalJSON() ([]byte, error) {
	if p == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(p))
}

// Context is the machine's extended state; the json tags are the JS context keys.
type Context struct {
	Board  []Player `json:"board"`
	Moves  int      `json:"moves"`
	Player Player   `json:"player"`
	Winner Player   `json:"winner,omitempty"`
}

// initialContext builds the JS `context` constant. Each call returns a fresh
// board, so no two snapshots share a backing array.
func initialContext() Context {
	return Context{Board: make([]Player, 9), Moves: 0, Player: X}
}

// Play builds the `{ type: 'PLAY', value }` event.
func Play(index int) xs.E {
	return xs.E{"type": "PLAY", "value": index}
}

// Reset builds the `{ type: 'RESET' }` event.
func Reset() xs.E { return xs.Ev("RESET") }

var winningLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

func other(p Player) Player {
	if p == X {
		return O
	}
	return X
}

// checkWin mirrors the guard of the same name: some winning line is all 'x' or all 'o'.
func checkWin(board []Player) bool {
	for _, line := range winningLines {
		for _, p := range [2]Player{X, O} {
			if board[line[0]] == p && board[line[1]] == p && board[line[2]] == p {
				return true
			}
		}
	}
	return false
}

// playIndex reads `event.value` as a board index. JS `board[value] === null` is
// false for any value that is not an index of the board (negative, fractional,
// too large, not a number), so ok=false maps to an invalid move. The value is an
// int when sent from Go and a float64 when replayed from a JSON golden file.
func playIndex(event xs.Event, size int) (int, bool) {
	e, isMap := event.(xs.E)
	if !isMap || e["type"] != "PLAY" {
		return 0, false
	}
	var f float64
	switch v := e["value"].(type) {
	case int:
		f = float64(v)
	case float64:
		f = v
	default:
		return 0, false
	}
	i := int(f)
	if float64(i) != f || i < 0 || i >= size {
		return 0, false
	}
	return i, true
}

// Machine mirrors ticTacToeMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actions: map[string]xs.Action{
			"updateBoard": xs.Assign(func(a xs.AssignArgs[Context]) Context {
				c := a.Context
				if a.Event.(xs.E)["type"] != "PLAY" {
					panic("Unexpected event type.")
				}
				i, _ := playIndex(a.Event, len(c.Board))
				board := append([]Player(nil), c.Board...)
				board[i] = c.Player
				c.Board = board
				c.Moves++
				c.Player = other(c.Player)
				return c
			}),
			"resetGame": xs.Assign(func(xs.AssignArgs[Context]) Context {
				return initialContext()
			}),
			"setWinner": xs.Assign(func(a xs.AssignArgs[Context]) Context {
				c := a.Context
				c.Winner = other(c.Player)
				return c
			}),
		},
		Guards: map[string]xs.Guard{
			"checkWin": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return checkWin(a.Context.Board)
			}),
			"checkDraw": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return a.Context.Moves == 9
			}),
			"isValidMove": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				i, ok := playIndex(a.Event, len(a.Context.Board))
				return ok && a.Context.Board[i] == ""
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		Initial: "playing",
		Context: initialContext(),
		States: xs.States{
			{
				Key: "playing",
				Always: xs.Transitions{
					{Target: "gameOver.winner", Guard: xs.GuardRef{Type: "checkWin"}},
					{Target: "gameOver.draw", Guard: xs.GuardRef{Type: "checkDraw"}},
				},
				On: map[string]xs.Transitions{
					"PLAY": {{
						Target:  "playing",
						Guard:   xs.GuardRef{Type: "isValidMove"},
						Actions: xs.Actions{xs.ActionRef{Type: "updateBoard"}},
					}},
				},
			},
			{
				Key:     "gameOver",
				Initial: "winner",
				States: xs.States{
					{Key: "winner", Tags: xs.Tags{"winner"}, Entry: xs.Actions{xs.ActionRef{Type: "setWinner"}}},
					{Key: "draw", Tags: xs.Tags{"draw"}},
				},
				On: map[string]xs.Transitions{
					"RESET": {{Target: "playing", Actions: xs.Actions{xs.ActionRef{Type: "resetGame"}}}},
				},
			},
		},
	})
}
