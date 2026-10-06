// Package tiles ports references/xstate/examples/tiles/src/tilesMachine.ts (a 4x4 sliding-tile game).
package tiles

import (
	"math"
	"math/rand"
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Tile mirrors the TS `Tile` interface: the board index plus its grid position.
type Tile struct {
	Index int `json:"index"`
	X     int `json:"x"`
	Y     int `json:"y"`
}

// Context is the machine's extended state. Selected and Hovered are nil where the
// JS context holds `undefined` (the key is then absent from the JSON snapshot).
type Context struct {
	Tiles    []int `json:"tiles"`
	Selected *Tile `json:"selected,omitempty"`
	Hovered  *Tile `json:"hovered,omitempty"`
}

// Event constructors for the machine's events, carrying the tile the way the UI does.

func SelectEvent(t Tile) xs.E { return xs.E{"type": "tile.select", "tile": t} }
func HoverEvent(t Tile) xs.E  { return xs.E{"type": "tile.hover", "tile": t} }
func MoveEvent() xs.E         { return xs.Ev("tile.move") }
func CancelEvent() xs.E       { return xs.Ev("move.canceled") }
func ShuffleEvent() xs.E      { return xs.Ev("shuffle") }

// tileOf reads event.tile. Events built in Go carry a Tile; events replayed from a
// JSON golden file carry map[string]any with float64 numbers.
func tileOf(e xs.Event) *Tile {
	ev, ok := e.(xs.E)
	if !ok {
		return nil
	}
	switch t := ev["tile"].(type) {
	case Tile:
		return &t
	case *Tile:
		return t
	case map[string]any:
		num := func(k string) int { f, _ := t[k].(float64); return int(f) }
		return &Tile{Index: num("index"), X: num("x"), Y: num("y")}
	}
	return nil
}

func rangeN(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func abs(n int) int { return int(math.Abs(float64(n))) }

// swap mirrors the TS swap helper but returns a copy: Go contexts are never mutated in place.
func swap(arr []int, a, b int) []int {
	out := slices.Clone(arr)
	out[a], out[b] = out[b], out[a]
	return out
}

// shuffle is the Fisher-Yates loop of the shuffleTiles action; random stands in for Math.random.
func shuffle(tiles []int, random func() float64) []int {
	out := slices.Clone(tiles)
	for i := len(out) - 1; i > 0; i-- {
		j := int(math.Floor(random() * float64(i+1)))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func assignCtx(fn func(c Context, e xs.Event) Context) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context { return fn(a.Context, a.Event) })
}

func act(name string) xs.Action { return xs.ActionRef{Type: name} }

// NewMachine mirrors tilesMachine. random stands in for Math.random in shuffleTiles.
func NewMachine(random func() float64) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Guards: map[string]xs.Guard{
			"isAdjacent": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				s, h := a.Context.Selected, a.Context.Hovered
				if s == nil || h == nil {
					return false
				}
				return (h.X == s.X && abs(h.Y-s.Y) == 1) || (h.Y == s.Y && abs(h.X-s.X) == 1)
			}),
			"allTilesInOrder": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				for idx, tile := range a.Context.Tiles {
					if tile != idx {
						return false
					}
				}
				return true
			}),
		},
		Actions: map[string]xs.Action{
			"clearSelectedTile": assignCtx(func(c Context, _ xs.Event) Context { c.Selected = nil; return c }),
			"clearHoveredTile":  assignCtx(func(c Context, _ xs.Event) Context { c.Hovered = nil; return c }),
			"setSelectedTile":   assignCtx(func(c Context, e xs.Event) Context { c.Selected = tileOf(e); return c }),
			"setHoveredTile":    assignCtx(func(c Context, e xs.Event) Context { c.Hovered = tileOf(e); return c }),
			"swapTiles": assignCtx(func(c Context, _ xs.Event) Context {
				c.Tiles = swap(c.Tiles, c.Hovered.Index, c.Selected.Index)
				return c
			}),
			"shuffleTiles": assignCtx(func(c Context, _ xs.Event) Context {
				c.Tiles = shuffle(c.Tiles, random)
				return c
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		Context: Context{Tiles: rangeN(16)},
		Initial: "start",
		States: xs.States{
			{Key: "start"},
			{
				Key: "gameOver",
				ID:  "gameOver",
				// make the game replayable
				On: map[string]xs.Transitions{
					"shuffle": {{Target: "playing", Actions: xs.Actions{act("shuffleTiles")}}},
				},
			},
			{
				Key: "playing",
				On: map[string]xs.Transitions{
					// targetless and actionless: swallows `shuffle` while a game is running
					"shuffle": {{}},
				},
				Initial: "selecting",
				States: xs.States{
					{
						Key: "selecting",
						ID:  "selecting",
						On: map[string]xs.Transitions{
							"tile.select": {{Target: "selected", Actions: xs.Actions{act("setSelectedTile")}}},
						},
					},
					{
						Key: "selected",
						On: map[string]xs.Transitions{
							"move.canceled": {{
								Actions: xs.Actions{act("clearSelectedTile"), act("clearHoveredTile")},
								Target:  "selecting",
							}},
							"tile.hover": {{Actions: xs.Actions{act("setHoveredTile")}}},
							"tile.move": {{
								Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[Context]) {
									if a.Check(xs.GuardRef{Type: "isAdjacent"}) {
										a.Enqueue(act("swapTiles"))
										a.Enqueue(act("clearSelectedTile"))
										a.Enqueue(act("clearHoveredTile"))
									}
								})},
								Target: "#selecting",
							}},
						},
					},
				},
				Always: xs.Transitions{{Guard: xs.GuardRef{Type: "allTilesInOrder"}, Target: "#gameOver"}},
			},
		},
		On: map[string]xs.Transitions{
			"shuffle": {{Target: ".playing", Actions: xs.Actions{act("shuffleTiles")}}},
		},
	})
}

// Machine mirrors tilesMachine with Math.random.
func Machine() *xs.StateMachine[Context] { return NewMachine(rand.Float64) }
