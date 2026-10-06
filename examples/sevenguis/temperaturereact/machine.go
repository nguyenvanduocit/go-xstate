// Package temperaturereact ports references/xstate/examples/7guis-temperature-react/src/temperatureMachine.ts.
package temperaturereact

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Number is a JS number: NaN and +-Infinity serialise as null, like JSON.stringify.
type Number float64

// MarshalJSON mirrors JSON.stringify for non-finite numbers.
func (n Number) MarshalJSON() ([]byte, error) {
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	return json.Marshal(f)
}

// Context mirrors TemperatureContext. Each field is nil (undefined, omitted from JSON),
// a string (the raw input) or a Number (the converted value).
type Context struct {
	TempC any `json:"tempC,omitempty"`
	TempF any `json:"tempF,omitempty"`
}

var decimalLiteral = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// isJSSpace reports WhiteSpace or LineTerminator of ECMA-262 (Zs, tab, VT, FF, LF, CR, LS, PS, BOM).
func isJSSpace(r rune) bool {
	return unicode.Is(unicode.Zs, r) || strings.ContainsRune("\t\n\v\f\r\u2028\u2029\uFEFF", r)
}

// toNumber mirrors the unary `+` operator on a string: StringToNumber of ECMA-262.
func toNumber(s string) float64 {
	s = strings.TrimFunc(s, isJSSpace)
	switch {
	case s == "":
		return 0
	case s == "Infinity" || s == "+Infinity":
		return math.Inf(1)
	case s == "-Infinity":
		return math.Inf(-1)
	case len(s) > 2 && s[0] == '0' && strings.ContainsRune("xXoObB", rune(s[1])):
		base := map[byte]int{'x': 16, 'o': 8, 'b': 2}[s[1]|0x20]
		if u, err := strconv.ParseUint(s[2:], base, 64); err == nil {
			return float64(u)
		}
		return math.NaN()
	case decimalLiteral.MatchString(s):
		f, _ := strconv.ParseFloat(s, 64) // out-of-range yields +-Inf, as in JS
		return f
	}
	return math.NaN()
}

func value(e xs.Event) string {
	v, _ := e.(xs.E)["value"].(string)
	return v
}

// Machine mirrors temperatureMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		Context: Context{},
		On: map[string]xs.Transitions{
			"CELSIUS": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				v := value(a.Event)
				var f any = ""
				if len(v) > 0 {
					// float64() blocks fused multiply-add so the result equals JS.
					f = Number(float64(toNumber(v)*(9.0/5)) + 32)
				}
				return Context{TempC: v, TempF: f}
			})}}},
			"FAHRENHEIT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				v := value(a.Event)
				var c any = ""
				if len(v) > 0 {
					c = Number(float64(toNumber(v)-32) * (5.0 / 9))
				}
				return Context{TempC: c, TempF: v}
			})}}},
		},
	})
}
