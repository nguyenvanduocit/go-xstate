// Package snake ports references/xstate/examples/snake-react/src/snakeMachine.ts: the snake game machine and
// its pure helpers. The React components (App.tsx, main.tsx) are not ported.
package snake

import (
	"math"
	"math/rand/v2"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Dir mirrors the JS `Dir` union: "Up" | "Left" | "Down" | "Right".
type Dir string

const (
	Up    Dir = "Up"
	Left  Dir = "Left"
	Down  Dir = "Down"
	Right Dir = "Right"
)

// TickInterval is the period of the real `ticks` actor (setInterval(..., 80) in JS).
const TickInterval = 80 * time.Millisecond

// Point mirrors `Point`.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// BodyPart mirrors `BodyPart = Point & { dir }`.
type BodyPart struct {
	X   int `json:"x"`
	Y   int `json:"y"`
	Dir Dir `json:"dir"`
}

// Snake mirrors `Snake`; index 0 is the head.
type Snake []BodyPart

// Context mirrors SnakeMachineContext; json tags equal the JS context keys.
type Context struct {
	Snake     Snake `json:"snake"`
	GridSize  Point `json:"gridSize"`
	Dir       Dir   `json:"dir"`
	Apple     Point `json:"apple"`
	Score     int   `json:"score"`
	HighScore int   `json:"highScore"`
}

// GameObject mirrors the JS GameObject union; Dir is empty for an apple (JS: undefined).
type GameObject struct {
	Type string `json:"type"` // "head" | "body" | "apple"
	Dir  Dir    `json:"dir,omitempty"`
}

// GameObjectAtPos mirrors getGamObjectAtPos. The bool is false where JS returns undefined.
func GameObjectAtPos(c Context, p Point) (GameObject, bool) {
	if isSamePos(pt(head(c.Snake)), p) {
		return GameObject{Type: "head", Dir: c.Dir}, true
	}
	if isSamePos(c.Apple, p) {
		return GameObject{Type: "apple"}, true
	}
	if part, ok := find(body(c.Snake), p); ok {
		return GameObject{Type: "body", Dir: part.Dir}, true
	}
	return GameObject{}, false
}

var oppositeDir = map[Dir]Dir{Up: Down, Down: Up, Left: Right, Right: Left}

func pt(b BodyPart) Point { return Point{b.X, b.Y} }

func isSamePos(p1, p2 Point) bool { return p1 == p2 }

func isOutsideGrid(grid, p Point) bool {
	return p.X < 0 || p.X >= grid.X || p.Y < 0 || p.Y >= grid.Y
}

func find(parts []BodyPart, p Point) (BodyPart, bool) {
	for _, pp := range parts {
		if isSamePos(pt(pp), p) {
			return pp, true
		}
	}
	return BodyPart{}, false
}

func head(s Snake) BodyPart { return s[0] }

func body(s Snake) []BodyPart { return s[1:] }

func newHead(old BodyPart, dir Dir) BodyPart {
	switch dir {
	case Up:
		return BodyPart{old.X, old.Y - 1, dir}
	case Down:
		return BodyPart{old.X, old.Y + 1, dir}
	case Left:
		return BodyPart{old.X - 1, old.Y, dir}
	default:
		return BodyPart{old.X + 1, old.Y, dir}
	}
}

func moveSnake(s Snake, dir Dir) Snake {
	out := make(Snake, 0, len(s))
	out = append(out, newHead(head(s), dir))
	return append(out, s[:len(s)-1]...)
}

func growSnake(s Snake) Snake {
	out := make(Snake, 0, len(s)+1)
	out = append(out, s...)
	return append(out, s[len(s)-1])
}

func randomGridPoint(random func() float64, grid Point) Point {
	x := int(math.Floor(random() * float64(grid.X)))
	y := int(math.Floor(random() * float64(grid.Y)))
	return Point{x, y}
}

func newApple(random func() float64, grid Point, ineligible []BodyPart) Point {
	p := randomGridPoint(random, grid)
	for {
		if _, hit := find(ineligible, p); !hit {
			return p
		}
		p = randomGridPoint(random, grid)
	}
}

func makeInitialSnake(grid Point) Snake {
	h := BodyPart{X: grid.X / 2, Y: grid.Y / 2, Dir: Right}
	return Snake{h, {h.X - 1, h.Y, h.Dir}, {h.X - 2, h.Y, h.Dir}}
}

func makeInitialApple(grid Point) Point {
	return Point{X: grid.X * 3 / 4, Y: grid.Y / 2}
}

// CreateInitialContext mirrors createInitialContext.
func CreateInitialContext() Context {
	grid := Point{X: 25, Y: 15}
	return Context{
		GridSize:  grid,
		Snake:     makeInitialSnake(grid),
		Apple:     makeInitialApple(grid),
		Score:     0,
		HighScore: 0,
		Dir:       Right,
	}
}

func assignCtx(fn func(c Context, e xs.Event) Context) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context { return fn(a.Context, a.Event) })
}

// Ticks mirrors the `ticks` actor: sends TICK to the parent every interval until stopped.
func Ticks(interval time.Duration) xs.ActorLogic {
	return xs.FromCallback(func(a xs.CallbackArgs) func() {
		t := time.NewTicker(interval)
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-t.C:
					a.SendBack(xs.Ev("TICK"))
				case <-done:
					return
				}
			}
		}()
		return func() {
			t.Stop()
			close(done)
		}
	})
}

// NewMachine mirrors snakeMachine. random stands in for Math.random (apple placement), and
// tickInterval for the 80 ms of the `ticks` actor.
func NewMachine(random func() float64, tickInterval time.Duration) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Guards: map[string]xs.Guard{
			"ate apple": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return isSamePos(pt(head(a.Context.Snake)), a.Context.Apple)
			}),
			"hit tail": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				_, hit := find(body(a.Context.Snake), pt(head(a.Context.Snake)))
				return hit
			}),
			"hit wall": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return isOutsideGrid(a.Context.GridSize, pt(head(a.Context.Snake)))
			}),
		},
		Actions: map[string]xs.Action{
			"move snake": assignCtx(func(c Context, _ xs.Event) Context {
				c.Snake = moveSnake(c.Snake, c.Dir)
				return c
			}),
			"save dir": assignCtx(func(c Context, e xs.Event) Context {
				if ev, ok := e.(xs.E); ok && ev.EventType() == "ARROW_KEY" {
					if d, _ := ev["dir"].(string); Dir(d) != oppositeDir[c.Dir] {
						c.Dir = Dir(d)
					}
				}
				return c
			}),
			"increase score": assignCtx(func(c Context, _ xs.Event) Context {
				c.Score++
				c.HighScore = max(c.Score, c.HighScore)
				return c
			}),
			"show new apple": assignCtx(func(c Context, _ xs.Event) Context {
				c.Apple = newApple(random, c.GridSize, c.Snake)
				return c
			}),
			"grow snake": assignCtx(func(c Context, _ xs.Event) Context {
				c.Snake = growSnake(c.Snake)
				return c
			}),
			"reset": assignCtx(func(c Context, _ xs.Event) Context {
				n := CreateInitialContext()
				n.HighScore = c.HighScore
				return n
			}),
		},
		Actors: map[string]xs.ActorLogic{"ticks": Ticks(tickInterval)},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "SnakeMachine",
		Context: CreateInitialContext(),
		Initial: "New Game",
		States: xs.States{
			{
				Key: "New Game",
				On: map[string]xs.Transitions{
					"ARROW_KEY": {{Actions: xs.Actions{xs.ActionRef{Type: "save dir"}}, Target: "Moving"}},
				},
			},
			{
				Key:    "Moving",
				Entry:  xs.Actions{xs.ActionRef{Type: "move snake"}},
				Invoke: []xs.InvokeConfig{{Src: "ticks"}},
				Always: xs.Transitions{
					{
						Guard: xs.GuardRef{Type: "ate apple"},
						Actions: xs.Actions{
							xs.ActionRef{Type: "grow snake"},
							xs.ActionRef{Type: "increase score"},
							xs.ActionRef{Type: "show new apple"},
						},
					},
					{
						Guard:  xs.Or(xs.GuardRef{Type: "hit tail"}, xs.GuardRef{Type: "hit wall"}),
						Target: "Game Over",
					},
				},
				On: map[string]xs.Transitions{
					"TICK":      {{Actions: xs.Actions{xs.ActionRef{Type: "move snake"}}}},
					"ARROW_KEY": {{Actions: xs.Actions{xs.ActionRef{Type: "save dir"}}, Target: "Moving"}},
				},
			},
			{
				Key: "Game Over",
				On: map[string]xs.Transitions{
					"NEW_GAME": {{
						Actions:     xs.Actions{xs.ActionRef{Type: "reset"}},
						Description: `triggered by pressing the "r" key`,
						Target:      "New Game",
					}},
				},
			},
		},
	})
}

// Machine mirrors snakeMachine with Math.random and the 80 ms ticks.
func Machine() *xs.StateMachine[Context] {
	return NewMachine(rand.Float64, TickInterval)
}
