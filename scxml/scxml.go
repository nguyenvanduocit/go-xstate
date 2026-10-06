// Package scxml converts SCXML documents into XState machines, mirroring
// packages/core/src/scxml.ts of XState JS.
package scxml

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// datamodel is the machine context: the SCXML datamodel, keyed by <data id>.
type datamodel = map[string]any

// ToMachine mirrors toMachine(xml): it parses an SCXML document and returns
// the equivalent machine. The context is the SCXML datamodel, keyed by
// <data id>. It panics on a document it cannot convert, as the JS version
// throws.
//
// Context values use nil for null and an internal value for undefined.
// JSON persistence reserves the exact object {"xstate$$type":"scxml.undefined"}
// to preserve undefined values across restoration.
func ToMachine(xml string) *xs.StateMachine[map[string]any] {
	doc, err := parseXML(xml)
	if err != nil {
		panic(err)
	}
	c := converter{js: newECMAScript()}
	return c.machine(doc)
}

// SanitizeStateID mirrors sanitizeStateId(id): it replaces every "." in an
// SCXML state id with "$" so the id is a valid XState state node id.
func SanitizeStateID(id string) string {
	return strings.ReplaceAll(id, ".", "$")
}

// converter carries the ECMAScript datamodel shared by a document and the
// machines it invokes.
type converter struct {
	js *ecmascript
}

// machine mirrors scxmlToMachine.
func (c converter) machine(doc *node) *xs.StateMachine[map[string]any] {
	root := doc.childrenNamed("scxml")[0]

	context := datamodel{}
	if dataModels := root.childrenNamed("datamodel"); len(dataModels) > 0 {
		for _, data := range dataModels[0].childrenNamed("data") {
			if src, _ := data.attr("src"); src != "" {
				panic(errors.New("Conversion of `src` attribute on datamodel's <data> elements is not supported."))
			}
			id, _ := data.attr("id")
			expr, ok := data.attr("expr")
			switch {
			case expr == "_sessionid":
				context[id] = undefinedValue{}
			case !ok:
				// eval('(undefined)')
				context[id] = undefinedValue{}
			default:
				context[id] = c.js.evalData(expr)
			}
		}
	}

	sc := c.stateConfig(root, "(machine)")
	return xs.CreateMachine(xs.MachineConfig[datamodel]{
		ID:      sc.ID,
		Context: context,
		Initial: sc.Initial,
		States:  sc.States,
		On:      sc.On,
		Always:  sc.Always,
		Entry:   sc.Entry,
		Exit:    sc.Exit,
		Invoke:  sc.Invoke,
	})
}

var (
	// whitespaceRe mirrors JS /\s+/, which also matches \v and Unicode spaces.
	whitespaceRe = regexp.MustCompile(`[\t\n\v\f\r\x{2028}\x{2029}\x{FEFF}\p{Zs}]+`)
	inRe         = regexp.MustCompile(`^In\('(.*)'\)`)
	notInRe      = regexp.MustCompile(`^!In\('(.*)'\)`)
	doneStateRe  = regexp.MustCompile(`^done\.state(\.|$)`)
	doneInvokeRe = regexp.MustCompile(`^done\.invoke(\.|$)`)
)

// stateConfig mirrors toConfig(nodeJson, id).
func (c converter) stateConfig(n *node, id string) xs.StateConfig {
	if n.name == "history" {
		history := xs.Shallow
		if t, _ := n.attr("type"); t == "deep" {
			history = xs.Deep
		}
		cfg := xs.StateConfig{Key: id, ID: id, Type: xs.History, History: history}
		if len(n.children) == 0 {
			return cfg
		}
		transition := n.childrenNamed("transition")[0]
		if target, _ := transition.attr("target"); target != "" {
			cfg.Target = "#" + SanitizeStateID(target)
		}
		return cfg
	}

	var stateType xs.StateType
	switch n.name {
	case "parallel":
		stateType = xs.Parallel
	case "final":
		stateType = xs.Final
	}

	if len(n.children) == 0 {
		// JS keeps only `type: 'final'` here; an empty <parallel/> is atomic.
		cfg := xs.StateConfig{Key: id, ID: id}
		if stateType == xs.Final {
			cfg.Type = xs.Final
		}
		return cfg
	}

	var initial string
	if stateType != xs.Parallel {
		initial, _ = n.attr("initial")
	}

	stateElements := n.childrenNamed("state", "parallel", "final", "history")

	if initial == "" {
		initialElements := n.childrenNamed("initial")
		switch {
		case len(initialElements) > 0:
			if len(initialElements[0].children) > 0 {
				initial, _ = initialElements[0].childrenNamed("transition")[0].attr("target")
			}
		case len(stateElements) > 0:
			initial, _ = stateElements[0].attr("id")
		}
	}

	cfg := xs.StateConfig{Key: id, ID: SanitizeStateID(id), Type: stateType}

	if initial != "" {
		resolved := strings.Split(initial, " ")
		if len(resolved) > 1 {
			panic(fmt.Errorf("Multiple initial states are not supported (%q).", initial))
		}
		cfg.Initial = SanitizeStateID(resolved[0])
	}

	for _, child := range c.indexedStates(stateElements) {
		cfg.States = append(cfg.States, c.stateConfig(child.node, child.key))
	}

	on := map[string]xs.Transitions{}
	var onKeys []string // insertion order of the JS `on` object
	for _, t := range n.childrenNamed("transition") {
		event, _ := t.attr("event")
		for _, eventType := range whitespaceRe.Split(event, -1) {
			transition := c.transitionConfig(t)
			if eventType == "" {
				cfg.Always = append(cfg.Always, transition)
				continue
			}
			eventType = doneEventType(eventType)
			if _, seen := on[eventType]; !seen {
				onKeys = append(onKeys, eventType)
			}
			on[eventType] = append(on[eventType], transition)
		}
	}
	// appendWildcards: "foo" and "foo.*" land on the same key, and the one
	// inserted later replaces the other, as Map.set does in JS.
	cfg.On = map[string]xs.Transitions{}
	for _, key := range onKeys {
		cfg.On[appendWildcard(key)] = on[key]
	}

	cfg.Entry = xs.Actions{}
	for _, onEntry := range n.childrenNamed("onentry") {
		cfg.Entry = append(cfg.Entry, c.actions(onEntry.children)...)
	}
	cfg.Exit = xs.Actions{}
	for _, onExit := range n.childrenNamed("onexit") {
		cfg.Exit = append(cfg.Exit, c.actions(onExit.children)...)
	}

	for _, invoke := range n.childrenNamed("invoke") {
		cfg.Invoke = append(cfg.Invoke, c.invokeConfig(invoke))
	}

	return cfg
}

type keyedNode struct {
	key  string
	node *node
}

// indexedStates mirrors indexedRecord(stateElements, (item) =>
// sanitizeStateId(`${item.attributes.id}`)): a later duplicate id replaces
// the earlier element but keeps its position, as a JS object key does.
func (c converter) indexedStates(elements []*node) []keyedNode {
	var out []keyedNode
	index := map[string]int{}
	for _, el := range elements {
		id, ok := el.attr("id")
		if !ok {
			id = "undefined"
		}
		key := SanitizeStateID(id)
		if i, seen := index[key]; seen {
			out[i].node = el
			continue
		}
		index[key] = len(out)
		out = append(out, keyedNode{key: key, node: el})
	}
	return out
}

// doneEventType mirrors toConfig's rewrite of SCXML done.state.* and
// done.invoke.* descriptors to the XState done event types.
func doneEventType(eventType string) string {
	switch {
	case doneStateRe.MatchString(eventType):
		return "xstate." + eventType
	case doneInvokeRe.MatchString(eventType):
		return "xstate.done.actor" + strings.TrimPrefix(eventType, "done.invoke")
	}
	return eventType
}

// appendWildcard mirrors appendWildcards for one descriptor: every
// descriptor except "*" and "foo.*" also matches its dot-separated
// descendants (SCXML prefix matching).
func appendWildcard(descriptor string) string {
	if descriptor != "*" && !strings.HasSuffix(descriptor, ".*") {
		return descriptor + ".*"
	}
	return descriptor
}

// jsAttr mirrors interpolating `${element.attributes.name}` into a JS
// function body: a missing attribute becomes the literal `undefined`.
func jsAttr(el *node, name string) string {
	if v, ok := el.attr(name); ok {
		return v
	}
	return "undefined"
}

// transitionConfig mirrors the transition object built for each event of a
// <transition> element.
func (c converter) transitionConfig(t *node) xs.TransitionConfig {
	var cfg xs.TransitionConfig
	if targets, _ := t.attr("target"); targets != "" {
		for _, target := range whitespaceRe.Split(targets, -1) {
			cfg.Targets = append(cfg.Targets, "#"+SanitizeStateID(target))
		}
	}
	if len(t.children) > 0 {
		cfg.Actions = c.actions(t.children)
	}
	if cond, _ := t.attr("cond"); cond != "" {
		cfg.Guard = c.transitionGuard(cond)
	}
	if typ, _ := t.attr("type"); typ != "internal" {
		cfg.Reenter = true
	}
	return cfg
}

// transitionGuard mirrors the guard of a <transition cond>: In('id') and
// !In('id') become stateIn guards; an In-prefixed cond that does not match
// those shapes gets no guard, as in JS.
func (c converter) transitionGuard(cond string) xs.Guard {
	switch {
	case strings.HasPrefix(cond, "In"):
		if m := inRe.FindStringSubmatch(strings.TrimSpace(cond)); m != nil {
			return xs.StateIn("#" + m[1])
		}
		return nil
	case strings.HasPrefix(cond, "!In"):
		if m := notInRe.FindStringSubmatch(strings.TrimSpace(cond)); m != nil {
			return xs.Not(xs.StateIn("#" + m[1]))
		}
		return nil
	}
	return c.guard(cond)
}

// guard mirrors createGuard(cond).
func (c converter) guard(cond string) xs.Guard {
	return xs.GuardFunc(func(a xs.GuardArgs[datamodel]) bool {
		return c.js.test(a.Context, a.Event, cond)
	})
}

// expr mirrors an action argument computed by evaluateExecutableContent.
func (c converter) expr(body string) xs.Expr {
	return xs.NewExpr(func(a xs.ExprArgs[datamodel]) any {
		return c.js.eval(a.Context, a.Event, body)
	})
}

// invokeConfig mirrors the conversion of an <invoke> element.
func (c converter) invokeConfig(el *node) xs.InvokeConfig {
	if typ, _ := el.attr("type"); typ != "scxml" && typ != "http://www.w3.org/TR/scxml/" {
		panic(errors.New("Currently only converting invoke elements of type SCXML is supported."))
	}
	cfg := xs.InvokeConfig{Logic: c.machine(el.childrenNamed("content")[0])}
	if id, _ := el.attr("id"); id != "" {
		cfg.ID = id
	}
	return cfg
}

// actions mirrors mapActions.
func (c converter) actions(elements []*node) xs.Actions {
	out := xs.Actions{}
	for _, el := range elements {
		out = append(out, c.action(el))
	}
	return out
}

// action mirrors mapAction.
func (c converter) action(el *node) xs.Action {
	switch el.name {
	case "raise":
		event, _ := el.attr("event")
		return xs.Raise(xs.Ev(event))
	case "assign":
		location, expr := jsAttr(el, "location"), jsAttr(el, "expr")
		body := fmt.Sprintf("\n%s;\n\nreturn {'%s': %s};", location, location, expr)
		return xs.Assign(func(a xs.AssignArgs[datamodel]) datamodel {
			update, _ := c.js.eval(a.Context, a.Event, body).(map[string]any)
			next := maps.Clone(a.Context)
			if next == nil {
				next = datamodel{}
			}
			maps.Copy(next, update)
			return next
		})
	case "cancel":
		if sendID, ok := el.attr("sendid"); ok {
			return xs.Cancel(sendID)
		}
		return xs.Cancel(c.expr("return " + jsAttr(el, "sendidexpr") + ";"))
	case "send":
		return c.send(el)
	case "log":
		expr := jsAttr(el, "expr")
		var label []string
		if l, ok := el.attr("label"); ok {
			label = []string{l}
		}
		return xs.Log(c.expr("return "+expr+";"), label...)
	case "if":
		return c.ifAction(el)
	}
	name := el.name
	if el.text {
		name = "undefined"
	}
	panic(fmt.Errorf("Conversion of %q elements is not implemented yet.", name))
}

// send mirrors the conversion of a <send> element.
func (c converter) send(el *node) xs.Action {
	event, hasEvent := el.attr("event")
	hasEvent = hasEvent && event != ""
	eventExpr, hasEventExpr := el.attr("eventexpr")

	var params string
	for _, child := range el.children {
		if child.name == "content" {
			panic(errors.New("Conversion of <content/> inside <send/> not implemented."))
		}
		params += jsAttr(child, "name") + ":" + jsAttr(child, "expr") + ",\n"
	}

	var convertedEvent any
	if hasEvent && params == "" {
		convertedEvent = xs.Ev(event)
	} else {
		// JS interpolates both attributes verbatim into the function body.
		typ := "undefined"
		if hasEvent {
			typ = `"` + event + `"`
		} else if hasEventExpr {
			typ = eventExpr
		}
		body := "return { type: " + typ + ", " + params + " }"
		convertedEvent = xs.NewExpr(func(a xs.ExprArgs[datamodel]) any {
			ev, _ := c.js.eval(a.Context, a.Event, body).(map[string]any)
			return xs.E(ev)
		})
	}

	var opts xs.SendOptions
	if delay, ok := el.attr("delay"); ok {
		if ms, defined := delayToMs(delay); defined {
			opts.Delay = ms
		}
	} else if delayExpr, _ := el.attr("delayexpr"); delayExpr != "" {
		body := "return " + delayExpr + ";"
		opts.Delay = xs.NewExpr(func(a xs.ExprArgs[datamodel]) any {
			ms, defined := delayToMs(c.js.eval(a.Context, a.Event, body))
			if !defined {
				return nil
			}
			return ms
		})
	}
	if id, ok := el.attr("id"); ok {
		opts.ID = id
	}

	target, hasTarget := el.attr("target")
	if target == "#_internal" {
		return xs.Raise(convertedEvent)
	}
	if hasTarget {
		return xs.SendTo(target, convertedEvent, opts)
	}
	return xs.SendTo(xs.NewExpr(func(a xs.ExprArgs[datamodel]) any { return a.Self }), convertedEvent, opts)
}

// ifAction mirrors the conversion of an <if>/<elseif>/<else> block into
// enqueueActions: the actions of the first branch whose guard passes.
func (c converter) ifAction(el *node) xs.Action {
	type branch struct {
		guard   xs.Guard
		actions xs.Actions
	}
	current := branch{guard: c.guard(jsAttr(el, "cond"))}
	var branches []branch
	for _, child := range el.children {
		switch child.name {
		case "elseif":
			branches = append(branches, current)
			current = branch{guard: c.guard(jsAttr(child, "cond"))}
		case "else":
			branches = append(branches, current)
			current = branch{}
		default:
			current.actions = append(current.actions, c.action(child))
		}
	}
	branches = append(branches, current)

	return xs.EnqueueActions(func(a xs.EnqueueArgs[datamodel]) {
		for _, b := range branches {
			if b.guard == nil || a.Check(b.guard) {
				for _, action := range b.actions {
					a.Enqueue(action)
				}
				break
			}
		}
	})
}

var (
	millisecondsRe = regexp.MustCompile(`(\d+)ms`)
	secondsRe      = regexp.MustCompile(`(\d*)(\.?)(\d+)s`)
)

// delayToMs mirrors delayToMs(delay): a number is milliseconds, a string is
// parsed as "<n>ms" or "<n>s"/"<n>.<m>s". defined is false for an empty
// delay (JS undefined). It panics on an unparsable delay, as JS throws.
func delayToMs(delay any) (ms time.Duration, defined bool) {
	switch d := delay.(type) {
	case nil, undefinedValue:
		return 0, false
	case int64:
		if d == 0 {
			return 0, false
		}
		return time.Duration(d) * time.Millisecond, true
	case float64:
		if d == 0 {
			return 0, false
		}
		return time.Duration(d * float64(time.Millisecond)), true
	case string:
		if d == "" {
			return 0, false
		}
		if m := millisecondsRe.FindStringSubmatch(d); m != nil {
			n, _ := strconv.Atoi(m[1])
			return time.Duration(n) * time.Millisecond, true
		}
		if m := secondsRe.FindStringSubmatch(d); m != nil {
			if m[2] == "" {
				n, _ := strconv.Atoi(m[3])
				return time.Duration(n) * time.Second, true
			}
			var seconds int
			if m[1] != "" {
				seconds, _ = strconv.Atoi(m[1])
			}
			frac := m[3]
			for len(frac) < 3 {
				frac += "0"
			}
			milliseconds, _ := strconv.Atoi(frac)
			if milliseconds >= 1000 {
				panic(fmt.Errorf("Can't parse \"%s delay.\"", d))
			}
			return time.Duration(seconds)*time.Second + time.Duration(milliseconds)*time.Millisecond, true
		}
	}
	panic(fmt.Errorf("Can't parse \"%v delay.\"", delay))
}
