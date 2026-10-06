// Package tictactoe ports references/xstate/examples/store-tic-tac-toe/src/store.ts.
package tictactoe

import (
	"encoding/json"
	"math"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Mark is a board cell, a player or an outcome winner. The empty Mark is JS
// null (an empty cell, no winner) and marshals to JSON null.
type Mark string

// Marks used by the game. Draw is only a Winner value.
const (
	X    Mark = "x"
	O    Mark = "o"
	Draw Mark = "draw"
)

// MarshalJSON writes the empty Mark as null.
func (m Mark) MarshalJSON() ([]byte, error) {
	if m == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(m))
}

// Status mirrors the JS `Status` union.
type Status string

// Game statuses. The JS `Status` union also has 'draw', which the store never
// produces (see the `played` handler).
const (
	Playing Status = "playing"
	Won     Status = "won"
)

// Context mirrors GameState. Board is a value array, so copying a Context
// copies the board.
type Context struct {
	Board         [9]Mark `json:"board"`
	CurrentPlayer Mark    `json:"currentPlayer"`
	Status        Status  `json:"status"`
}

// InitialState mirrors the module-level `initialState`.
var InitialState = Context{CurrentPlayer: X, Status: Playing}

// Outcome mirrors GameOutcome: Winner is x, o, draw or empty (null); Line is
// the winning line or nil (null).
type Outcome struct {
	Winner Mark  `json:"winner"`
	Line   []int `json:"line"`
}

var lines = [8][3]int{
	{0, 1, 2},
	{3, 4, 5},
	{6, 7, 8},
	{0, 3, 6},
	{1, 4, 7},
	{2, 5, 8},
	{0, 4, 8},
	{2, 4, 6},
}

// GetGameOutcome mirrors getGameOutcome.
func GetGameOutcome(board [9]Mark) Outcome {
	for _, line := range lines {
		a, b, c := line[0], line[1], line[2]
		if board[a] != "" && board[a] == board[b] && board[a] == board[c] {
			return Outcome{Winner: board[a], Line: []int{a, b, c}}
		}
	}
	full := true
	for _, cell := range board {
		if cell == "" {
			full = false
			break
		}
	}
	if full {
		return Outcome{Winner: Draw}
	}
	return Outcome{}
}

// position reads event.position. JS indexes `board[event.position]`, which is
// undefined (never null) for a missing, fractional or out-of-range position;
// ok=false covers all of those. Ints come from Go callers, float64 from
// JSON-decoded traces.
func position(ev xs.Event) (pos int, ok bool) {
	var f float64
	switch v := ev.(xs.E)["position"].(type) {
	case int:
		f = float64(v)
	case float64:
		f = v
	default:
		return 0, false
	}
	if f != math.Trunc(f) || f < 0 || f >= 9 {
		return 0, false
	}
	return int(f), true
}

// Config mirrors the object passed to createStore in store.ts.
func Config() xstore.StoreConfig[Context] {
	return xstore.StoreConfig[Context]{
		Context: InitialState,
		On: map[string]xstore.StoreAssigner[Context]{
			"played": func(c Context, ev xs.Event, _ *xstore.EnqueueObject[Context]) (Context, bool) {
				pos, ok := position(ev)
				// Ignore moves if cell is already taken or game is not in playing state
				if c.Status != Playing || !ok || c.Board[pos] != "" {
					return c, true
				}

				newBoard := c.Board
				newBoard[pos] = c.CurrentPlayer

				outcome := GetGameOutcome(newBoard)
				nextPlayer := X
				if c.CurrentPlayer == X {
					nextPlayer = O
				}
				// The JS test is `outcome.winner ? 'won' : outcome.winner === 'draw'
				// ? 'draw' : 'playing'`; 'draw' is truthy, so a draw yields 'won' and
				// the 'draw' branch never runs. Ported as is.
				nextStatus := Playing
				if outcome.Winner != "" {
					nextStatus = Won
				}

				return Context{Board: newBoard, CurrentPlayer: nextPlayer, Status: nextStatus}, true
			},
			"reset": func(Context, xs.Event, *xstore.EnqueueObject[Context]) (Context, bool) {
				return InitialState, true
			},
		},
	}
}

// Game mirrors what App.tsx reads from the module-level store: the store and
// the selectors passed to useSelector (context.board, context.currentPlayer,
// context.status).
type Game struct {
	Store         *xstore.Store[Context]
	Board         xstore.ReadonlyAtom[[9]Mark]
	CurrentPlayer xstore.ReadonlyAtom[Mark]
	Status        xstore.ReadonlyAtom[Status]
}

// NewGame mirrors `export const gameStore = createStore(...)` plus the
// selectors of App.tsx.
func NewGame() *Game {
	s := xstore.CreateStore(Config())
	return &Game{
		Store:         s,
		Board:         xstore.Select(s, func(c Context) [9]Mark { return c.Board }),
		CurrentPlayer: xstore.Select(s, func(c Context) Mark { return c.CurrentPlayer }),
		Status:        xstore.Select(s, func(c Context) Status { return c.Status }),
	}
}
