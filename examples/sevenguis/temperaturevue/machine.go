// Package temperaturevue ports references/xstate/examples/7guis-2-temperature-vue/src/tempMachine.ts.
package temperaturevue

import (
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Num is a JS number. Like JSON.stringify, it serialises NaN and +-Infinity as
// null, so snapshots compare equal to the JS ones.
type Num float64

// MarshalJSON mirrors JSON.stringify for numbers.
func (n Num) MarshalJSON() ([]byte, error) {
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	return json.Marshal(f + 0) // +0 turns -0 into 0, as JS does
}

// Context is the converter's extended state. A nil field is JS `undefined`
// (omitted from the serialised snapshot).
type Context struct {
	Celsius    *Num `json:"celsius,omitempty"`
	Fahrenheit *Num `json:"fahrenheit,omitempty"`
}

func num(f float64) *Num { n := Num(f); return &n }

func eventValue(e xs.Event) string { return e.(xs.E)["value"].(string) }

// isJSSpace reports whether r is JS WhiteSpace or LineTerminator
// (String.prototype.trim and the StringToNumber grammar).
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

func jsTrim(s string) string { return strings.TrimFunc(s, isJSSpace) }

func isOnlyWhiteSpace(s string) bool { return len(jsTrim(s)) == 0 }

var (
	decimalLiteral = regexp.MustCompile(`^[+-]?(?:[0-9]+\.?[0-9]*|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
	radixLiteral   = regexp.MustCompile(`^0([xX][0-9a-fA-F]+|[oO][0-7]+|[bB][01]+)$`)
)

// toNumber mirrors the JS unary plus on a string (StringToNumber).
func toNumber(s string) float64 {
	s = jsTrim(s)
	switch {
	case s == "":
		return 0
	case s == "Infinity" || s == "+Infinity":
		return math.Inf(1)
	case s == "-Infinity":
		return math.Inf(-1)
	case radixLiteral.MatchString(s):
		base := map[byte]int{'x': 16, 'X': 16, 'o': 8, 'O': 8, 'b': 2, 'B': 2}[s[1]]
		n, _ := new(big.Int).SetString(s[2:], base)
		f, _ := new(big.Float).SetInt(n).Float64()
		return f
	case decimalLiteral.MatchString(s):
		f, _ := strconv.ParseFloat(s, 64) // out-of-range yields +-Inf, as in JS
		return f
	}
	return math.NaN()
}

// jsRound mirrors Math.round: ties round toward +Infinity.
func jsRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	f := math.Floor(x)
	if x-f >= 0.5 {
		return f + 1
	}
	return f
}

// Machine mirrors tempMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Guards: map[string]xs.Guard{
			"valueIsNumber": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return !math.IsNaN(toNumber(eventValue(a.Event)))
			}),
		},
		Actions: map[string]xs.Action{
			"onChangeC": xs.Assign(func(a xs.AssignArgs[Context]) Context {
				v := eventValue(a.Event)
				if isOnlyWhiteSpace(v) {
					return Context{}
				}
				c := toNumber(v)
				const ratio = 9.0 / 5
				return Context{Celsius: num(c), Fahrenheit: num(jsRound(float64(c*ratio) + 32))}
			}),
			"onChangeF": xs.Assign(func(a xs.AssignArgs[Context]) Context {
				v := eventValue(a.Event)
				if isOnlyWhiteSpace(v) {
					return Context{}
				}
				f := toNumber(v)
				const ratio = 5.0 / 9
				return Context{Celsius: num(jsRound(float64((f - 32) * ratio))), Fahrenheit: num(f)}
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "tempConverter",
		Initial: "ready",
		Context: Context{},
		States: xs.States{
			{Key: "ready", On: map[string]xs.Transitions{
				"changeC": {{
					Target:  "ready",
					Guard:   xs.GuardRef{Type: "valueIsNumber"},
					Actions: xs.Actions{xs.ActionRef{Type: "onChangeC"}},
				}},
				"changeF": {{
					Target:  "ready",
					Guard:   xs.GuardRef{Type: "valueIsNumber"},
					Actions: xs.Actions{xs.ActionRef{Type: "onChangeF"}},
				}},
			}},
		},
	})
}
