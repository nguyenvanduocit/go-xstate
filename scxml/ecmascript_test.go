package scxml_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nguyenvanduocit/go-xstate/scxml"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func TestECMAScriptPreservesNullAndUndefined(t *testing.T) {
	for _, test := range []struct {
		name, data, entry, cond string
	}{
		{"null data", `<data id="x" expr="null"/>`, "", "x === null"},
		{"unbound data", `<data id="x"/>`, "", "x === undefined"},
		{"undefined data", `<data id="x" expr="undefined"/>`, "", "x === undefined"},
		{"sparse array", `<data id="x" expr="[, null]"/>`, "", "x[0] === undefined &amp;&amp; x[1] === null"},
		{"typed array", `<data id="x" expr="new Uint8Array([1, 2])"/>`, "", "x.length === 2 &amp;&amp; x[0] === 1 &amp;&amp; x[1] === 2"},
		{"nested typed array", `<data id="x" expr="({a:new Uint8Array([1, 2]), n:null})"/>`, "", "x.a.length === 2 &amp;&amp; x.a[0] === 1 &amp;&amp; x.n === null"},
		{"nested data", `<data id="x" expr="({n:null, u:undefined, a:[null, undefined, {n:null}]})"/>`, "", "x.n === null &amp;&amp; x.u === undefined &amp;&amp; x.a[0] === null &amp;&amp; x.a[1] === undefined &amp;&amp; x.a[2].n === null"},
		{"null assignment", `<data id="x" expr="1"/>`, `<assign location="x" expr="null"/>`, "x === null"},
		{"undefined assignment", `<data id="x" expr="null"/>`, `<assign location="x" expr="undefined"/>`, "x === undefined"},
		{"nested assignment", `<data id="x"/>`, `<assign location="x" expr="({n:null, a:[undefined, null]})"/>`, "x.n === null &amp;&amp; x.a[0] === undefined &amp;&amp; x.a[1] === null"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := fmt.Sprintf(`<scxml initial="a"><datamodel>%s</datamodel><state id="a"><onentry>%s</onentry><transition event="GO" cond="%s" target="b"/></state><final id="b"/></scxml>`, test.data, test.entry, test.cond)
			actor := xs.CreateActor(scxml.ToMachine(source)).Start()
			defer actor.Stop()
			actor.Send(xs.Ev("GO"))
			if snapshot := actor.GetSnapshot(); !snapshot.Matches("b") {
				t.Fatalf("guard did not match: state=%v context=%#v error=%v", snapshot.Value, snapshot.Context, snapshot.Error)
			}
		})
	}
}

func TestECMAScriptPreservesNullAndUndefinedInSentEvents(t *testing.T) {
	machine := scxml.ToMachine(`<scxml initial="a"><state id="a"><onentry>
		<send event="GO" target="#_internal">
			<param name="n" expr="null"/>
			<param name="u" expr="undefined"/>
			<param name="nested" expr="({a:[null, undefined]})"/>
		</send>
	</onentry><transition event="GO" cond="_event.data.n === null &amp;&amp; _event.data.u === undefined &amp;&amp; _event.data.nested.a[0] === null &amp;&amp; _event.data.nested.a[1] === undefined" target="b"/></state><final id="b"/></scxml>`)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	if snapshot := actor.GetSnapshot(); !snapshot.Matches("b") {
		t.Fatalf("event guard did not match: state=%v error=%v", snapshot.Value, snapshot.Error)
	}
}

func TestECMAScriptTreatsGoNilEventDataAsNull(t *testing.T) {
	machine := scxml.ToMachine(`<scxml initial="a"><state id="a"><transition event="GO" cond="_event.data.n === null &amp;&amp; _event.data.nested[0] === null &amp;&amp; _event.data.missing === undefined" target="b"/></state><final id="b"/></scxml>`)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	actor.Send(xs.E{"type": "GO", "n": nil, "nested": []any{nil}})
	if snapshot := actor.GetSnapshot(); !snapshot.Matches("b") {
		t.Fatalf("event guard did not match: state=%v error=%v", snapshot.Value, snapshot.Error)
	}
}

func TestECMAScriptRestoresNullAndUndefined(t *testing.T) {
	machine := scxml.ToMachine(`<scxml initial="a"><datamodel>
		<data id="unbound"/>
		<data id="explicit" expr="undefined"/>
		<data id="n" expr="null"/>
		<data id="nested" expr="({a:[null, undefined]})"/>
	</datamodel><state id="a"><transition event="GO" cond="unbound === undefined &amp;&amp; explicit === undefined &amp;&amp; n === null &amp;&amp; nested.a[0] === null &amp;&amp; nested.a[1] === undefined" target="b"/></state><final id="b"/></scxml>`)
	original := xs.CreateActor(machine).Start()
	defer original.Stop()
	persisted := original.GetPersistedSnapshot()
	encoded, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		snapshot any
	}{
		{"in memory", persisted},
		{"JSON decoded", decoded},
	} {
		t.Run(test.name, func(t *testing.T) {
			restored := xs.CreateActor(machine, xs.WithSnapshot(test.snapshot)).Start()
			defer restored.Stop()
			restored.Send(xs.Ev("GO"))
			if snapshot := restored.GetSnapshot(); !snapshot.Matches("b") {
				t.Fatalf("restored guard did not match: state=%v context=%#v error=%v", snapshot.Value, snapshot.Context, snapshot.Error)
			}
		})
	}
}
