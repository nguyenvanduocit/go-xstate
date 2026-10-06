package scxml_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/scxml"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type scxmlMachine = *xs.StateMachine[map[string]any]

type scxmlSnapshot = *xs.MachineSnapshot[map[string]any]

// scxmlCase is one (testGroupName, testName) pair of `testGroups` in
// scxml.test.ts. skip holds the JS comment of a case that is commented out
// there.
type scxmlCase struct {
	group string
	name  string
	skip  string
}

// scxmlCases mirrors `testGroups` in source order, commented-out entries
// included. Fixtures live in testdata/<group>/<name>.{scxml,json}, copied
// from @scion-scxml/test-framework. The `overrides` entry of the JS file
// (assign-current-small-step/test0.scxml) is copied from xstate's
// test/fixtures/scxml instead; its .json still comes from scion, as in JS.
var scxmlCases = []scxmlCase{
	{group: "actionSend", name: "send1"},
	{group: "actionSend", name: "send2"},
	{group: "actionSend", name: "send3"},
	{group: "actionSend", name: "send4"},
	{group: "actionSend", name: "send4b"},
	{group: "actionSend", name: "send7"},
	{group: "actionSend", name: "send7b"},
	{group: "actionSend", name: "send8"},
	{group: "actionSend", name: "send8b"},
	{group: "actionSend", name: "send9"},
	{group: "assign", name: "assign_invalid", skip: "this has a syntax error on purpose, so it's not included"},
	{group: "assign", name: "assign_obj_literal", skip: "deep initial states are not supported"},
	{group: "assign-current-small-step", name: "test0"},
	{group: "assign-current-small-step", name: "test1"},
	{group: "assign-current-small-step", name: "test2"},
	{group: "assign-current-small-step", name: "test3"},
	{group: "assign-current-small-step", name: "test4"},
	{group: "basic", name: "basic0"},
	{group: "basic", name: "basic1"},
	{group: "basic", name: "basic2"},
	{group: "cond-js", name: "test0"},
	{group: "cond-js", name: "test1"},
	{group: "cond-js", name: "test2"},
	{group: "cond-js", name: "TestConditionalTransition"},
	{group: "data", name: "data_invalid", skip: "commented out in JS without a reason"},
	{group: "data", name: "data_obj_literal", skip: "deep initial states are not supported"},
	{group: "default-initial-state", name: "initial1"},
	{group: "default-initial-state", name: "initial2"},
	{group: "delayedSend", name: "send1"},
	{group: "delayedSend", name: "send2"},
	{group: "delayedSend", name: "send3"},
	{group: "documentOrder", name: "documentOrder0"},
	{group: "error", name: "error", skip: "not implemented"},
	{group: "forEach", name: "test1", skip: "not implemented"},
	{group: "hierarchy", name: "hier0"},
	{group: "hierarchy", name: "hier1"},
	{group: "hierarchy", name: "hier2"},
	{group: "hierarchy+documentOrder", name: "test0"},
	{group: "hierarchy+documentOrder", name: "test1"},
	{group: "history", name: "history0"},
	{group: "history", name: "history1"},
	{group: "history", name: "history2"},
	{group: "history", name: "history3"},
	{group: "history", name: "history4"},
	{group: "history", name: "history4b"},
	{group: "history", name: "history5"},
	{group: "history", name: "history6"},
	{group: "if-else", name: "test0"},
	{group: "in", name: "TestInPredicate"},
	{group: "internal-transitions", name: "test0"},
	{group: "internal-transitions", name: "test1"},
	{group: "misc", name: "deep-initial", skip: "deep initial states are not supported"},
	{group: "more-parallel", name: "test0"},
	{group: "more-parallel", name: "test1"},
	{group: "more-parallel", name: "test2"},
	{group: "more-parallel", name: "test2b"},
	{group: "more-parallel", name: "test3"},
	{group: "more-parallel", name: "test3b"},
	{group: "more-parallel", name: "test4"},
	{group: "more-parallel", name: "test5"},
	{group: "more-parallel", name: "test6"},
	{group: "more-parallel", name: "test6b"},
	{group: "more-parallel", name: "test7"},
	{group: "more-parallel", name: "test8"},
	{group: "more-parallel", name: "test9"},
	{group: "more-parallel", name: "test10"},
	{group: "more-parallel", name: "test10b"},
	{group: "multiple-events-per-transition", name: "test1"},
	{group: "parallel", name: "test0"},
	{group: "parallel", name: "test1"},
	{group: "parallel", name: "test2"},
	{group: "parallel", name: "test3"},
	{group: "parallel+interrupt", name: "test0"},
	{group: "parallel+interrupt", name: "test1"},
	{group: "parallel+interrupt", name: "test2"},
	{group: "parallel+interrupt", name: "test3"},
	{group: "parallel+interrupt", name: "test4"},
	{group: "parallel+interrupt", name: "test5"},
	{group: "parallel+interrupt", name: "test6"},
	{group: "parallel+interrupt", name: "test7"},
	{group: "parallel+interrupt", name: "test7b"},
	{group: "parallel+interrupt", name: "test8"},
	{group: "parallel+interrupt", name: "test9"},
	{group: "parallel+interrupt", name: "test10"},
	{group: "parallel+interrupt", name: "test11"},
	{group: "parallel+interrupt", name: "test12"},
	{group: "parallel+interrupt", name: "test13"},
	{group: "parallel+interrupt", name: "test14"},
	{group: "parallel+interrupt", name: "test15"},
	{group: "parallel+interrupt", name: "test16"},
	{group: "parallel+interrupt", name: "test17"},
	{group: "parallel+interrupt", name: "test18"},
	{group: "parallel+interrupt", name: "test19"},
	{group: "parallel+interrupt", name: "test20"},
	{group: "parallel+interrupt", name: "test21"},
	{group: "parallel+interrupt", name: "test21b"},
	{group: "parallel+interrupt", name: "test21c"},
	{group: "parallel+interrupt", name: "test22"},
	{group: "parallel+interrupt", name: "test23"},
	{group: "parallel+interrupt", name: "test24"},
	{group: "parallel+interrupt", name: "test25"},
	{group: "parallel+interrupt", name: "test27"},
	{group: "parallel+interrupt", name: "test28"},
	{group: "parallel+interrupt", name: "test29"},
	{group: "parallel+interrupt", name: "test30"},
	{group: "parallel+interrupt", name: "test31"},
	{group: "script", name: "test0", skip: "<script/> conversion not implemented"},
	{group: "script", name: "test1", skip: "<script/> conversion not implemented"},
	{group: "script", name: "test2", skip: "<script/> conversion not implemented"},
	{group: "script-src", name: "test0", skip: "<script/> conversion not implemented"},
	{group: "script-src", name: "test1", skip: "<script/> conversion not implemented"},
	{group: "script-src", name: "test2", skip: "<script/> conversion not implemented"},
	{group: "script-src", name: "test3", skip: "<script/> conversion not implemented"},
	{group: "scxml-prefix-event-name-matching", name: "star0", skip: "this relies on the source order of transitions where * is first and it's supposed to get matched over an explicit descriptor"},
	{group: "scxml-prefix-event-name-matching", name: "test0", skip: "prefix event matching not implemented yet"},
	{group: "scxml-prefix-event-name-matching", name: "test1", skip: "prefix event matching not implemented yet"},
	{group: "send-data", name: "send1", skip: "<content> conversion not implemented"},
	{group: "send-idlocation", name: "test0", skip: "commented out in JS without a reason"},
	{group: "send-internal", name: "test0", skip: "commented out in JS without a reason"},
	{group: "targetless-transition", name: "test0"},
	{group: "targetless-transition", name: "test1"},
	{group: "targetless-transition", name: "test2"},
	{group: "targetless-transition", name: "test3"},
	{group: "w3c-ecma", name: "test144.txml"},
	{group: "w3c-ecma", name: "test147.txml"},
	{group: "w3c-ecma", name: "test148.txml"},
	{group: "w3c-ecma", name: "test149.txml"},
	{group: "w3c-ecma", name: "test150.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test151.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test152.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test153.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test155.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test156.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test158.txml"},
	{group: "w3c-ecma", name: "test159.txml", skip: "different error handling"},
	{group: "w3c-ecma", name: "test172.txml"},
	{group: "w3c-ecma", name: "test173.txml"},
	{group: "w3c-ecma", name: "test174.txml"},
	{group: "w3c-ecma", name: "test175.txml"},
	{group: "w3c-ecma", name: "test176.txml"},
	{group: "w3c-ecma", name: "test179.txml", skip: "conversion of <content> in <send> not implemented yet"},
	{group: "w3c-ecma", name: "test183.txml", skip: "idlocation not implemented yet"},
	{group: "w3c-ecma", name: "test185.txml"},
	{group: "w3c-ecma", name: "test186.txml"},
	{group: "w3c-ecma", name: "test187.txml"},
	{group: "w3c-ecma", name: "test189.txml"},
	{group: "w3c-ecma", name: "test190.txml"}, // JS note: _sessionid is undefined for expressions
	{group: "w3c-ecma", name: "test191.txml"},
	{group: "w3c-ecma", name: "test192.txml"},
	{group: "w3c-ecma", name: "test193.txml"},
	{group: "w3c-ecma", name: "test194.txml", skip: "it's using an invalid event target (another actor), we should be erroring on this somehow when we revamp the error story"},
	{group: "w3c-ecma", name: "test198.txml", skip: "origintype not implemented yet"},
	{group: "w3c-ecma", name: "test199.txml", skip: "send type not checked"},
	{group: "w3c-ecma", name: "test200.txml"},
	{group: "w3c-ecma", name: "test201.txml"},
	{group: "w3c-ecma", name: "test205.txml"},
	{group: "w3c-ecma", name: "test207.txml", skip: "delayexpr"},
	{group: "w3c-ecma", name: "test208.txml"},
	{group: "w3c-ecma", name: "test210.txml", skip: "sendidexpr not supported yet"},
	{group: "w3c-ecma", name: "test215.txml", skip: "<invoke typeexpr=\"...\">"},
	{group: "w3c-ecma", name: "test216.txml", skip: "<invoke srcexpr=\"...\">"},
	{group: "w3c-ecma", name: "test220.txml"},
	{group: "w3c-ecma", name: "test223.txml", skip: "idlocation not implemented yet"},
	{group: "w3c-ecma", name: "test224.txml", skip: "<invoke idlocation=\"...\">"},
	{group: "w3c-ecma", name: "test225.txml", skip: "unique invokeids generated at invoke time"},
	{group: "w3c-ecma", name: "test226.txml", skip: "<invoke src=\"...\">"},
	{group: "w3c-ecma", name: "test228.txml", skip: "this test relies on `invokeid` being available on the event"},
	{group: "w3c-ecma", name: "test229.txml", skip: "autoForward not supported in v5"},
	{group: "w3c-ecma", name: "test230.txml", skip: "autoForward not supported in v5"},
	{group: "w3c-ecma", name: "test232.txml"},
	{group: "w3c-ecma", name: "test233.txml", skip: "<finalize> not implemented yet"},
	{group: "w3c-ecma", name: "test234.txml", skip: "<finalize> not implemented yet"},
	{group: "w3c-ecma", name: "test235.txml"},
	{group: "w3c-ecma", name: "test236.txml", skip: "reaching a final state should execute all onexit handlers"},
	{group: "w3c-ecma", name: "test237.txml"},
	{group: "w3c-ecma", name: "test239.txml", skip: "<invoke src=\"...\">"},
	{group: "w3c-ecma", name: "test240.txml", skip: "conversion of namelist not implemented yet"},
	{group: "w3c-ecma", name: "test241.txml", skip: "conversion of namelist not implemented yet"},
	{group: "w3c-ecma", name: "test242.txml", skip: "<invoke src=\"...\">"},
	{group: "w3c-ecma", name: "test243.txml", skip: "conversion of <param> in <scxml> not implemented yet"},
	{group: "w3c-ecma", name: "test244.txml", skip: "conversion of namelist not implemented yet"},
	{group: "w3c-ecma", name: "test245.txml", skip: "conversion of namelist not implemented yet"},
	{group: "w3c-ecma", name: "test247.txml"},
	{group: "w3c-ecma", name: "test250.txml", skip: "this is a manual test - we could test it by snapshotting logged valued"},
	{group: "w3c-ecma", name: "test252.txml", skip: "this expects the parent to not receive the event sent from the canceled child's exit action"},
	{group: "w3c-ecma", name: "test253.txml", skip: "_event.origintype not implemented yet"},
	{group: "w3c-ecma", name: "test276.txml", skip: "<invoke src=\"...\">"},
	{group: "w3c-ecma", name: "test277.txml", skip: "illegal expression in datamodel creates unbound variable"},
	{group: "w3c-ecma", name: "test278.txml", skip: "non-root datamodel with early binding not implemented yet"},
	{group: "w3c-ecma", name: "test279.txml", skip: "non-root datamodel with early binding not implemented yet"},
	{group: "w3c-ecma", name: "test280.txml", skip: "non-root datamodel with late binding not implemented yet"},
	{group: "w3c-ecma", name: "test286.txml", skip: "this intentionally throws when executing assign, we should be erroring on this somehow when we revamp the error story"},
	{group: "w3c-ecma", name: "test287.txml"},
	{group: "w3c-ecma", name: "test294.txml", skip: "conversion of <donedata> not implemented yet"},
	{group: "w3c-ecma", name: "test298.txml", skip: "error.execution when evaluating donedata"},
	{group: "w3c-ecma", name: "test302.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test303-1.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test303-2.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test303.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test304.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test307.txml", skip: "non-root datamodel with late binding not implemented yet"},
	{group: "w3c-ecma", name: "test309.txml", skip: "error in cond expression being treated as false"},
	{group: "w3c-ecma", name: "test310.txml", skip: "conversion of In() predicate not implemented yet"},
	{group: "w3c-ecma", name: "test311.txml", skip: "error.execution when evaluating assign"},
	{group: "w3c-ecma", name: "test312.txml", skip: "error.execution when evaluating assign"},
	{group: "w3c-ecma", name: "test313.txml", skip: "error.execution when evaluating assign"},
	{group: "w3c-ecma", name: "test314.txml", skip: "error.execution when evaluating assign"},
	{group: "w3c-ecma", name: "test318.txml"},
	{group: "w3c-ecma", name: "test319.txml", skip: "SCXML has no init event, so _event stays unbound in onentry of initial state"},
	{group: "w3c-ecma", name: "test321.txml", skip: "_sessionid not yet available for expressions"},
	{group: "w3c-ecma", name: "test322.txml", skip: "_sessionid not yet available for expressions"},
	{group: "w3c-ecma", name: "test323.txml", skip: "_name not yet available for expressions"},
	{group: "w3c-ecma", name: "test324.txml", skip: "_name not yet available for expressions"},
	{group: "w3c-ecma", name: "test325.txml", skip: "_ioprocessors not yet available for expressions"},
	{group: "w3c-ecma", name: "test326.txml", skip: "_ioprocessors not yet available for expressions"},
	{group: "w3c-ecma", name: "test329.txml", skip: "system variables can't be modified, we don't keep them in datamodel, so it might be hard to run this test"},
	{group: "w3c-ecma", name: "test330.txml", skip: "SCXML _event properties not implemented yet"},
	{group: "w3c-ecma", name: "test331.txml", skip: "_event.type not implemented yet correctly"},
	{group: "w3c-ecma", name: "test332.txml", skip: "idlocation not implemented yet"},
	{group: "w3c-ecma", name: "test333.txml"},
	{group: "w3c-ecma", name: "test335.txml"},
	{group: "w3c-ecma", name: "test336.txml"},
	{group: "w3c-ecma", name: "test337.txml"},
	{group: "w3c-ecma", name: "test338.txml", skip: "<invoke idlocation=\"...\"> + _event.invokeid available on <send> events received from the invoked child"},
	{group: "w3c-ecma", name: "test339.txml"},
	{group: "w3c-ecma", name: "test342.txml"},
	{group: "w3c-ecma", name: "test343.txml", skip: "error.execution when evaluating donedata"},
	{group: "w3c-ecma", name: "test344.txml", skip: "error in cond expression being treated as false and raises error.execution"},
	{group: "w3c-ecma", name: "test346.txml", skip: "system variables can't be modified, we don't keep them in datamodel, so it might be hard to run this test"},
	{group: "w3c-ecma", name: "test347.txml"},
	{group: "w3c-ecma", name: "test348.txml"},
	{group: "w3c-ecma", name: "test349.txml"},
	{group: "w3c-ecma", name: "test350.txml", skip: "_sessionid not yet available for expressions"},
	{group: "w3c-ecma", name: "test351.txml", skip: "_event.sendid not implemented yet"},
	{group: "w3c-ecma", name: "test352.txml", skip: "_event.origintype not implemented yet"},
	{group: "w3c-ecma", name: "test354.txml", skip: "conversion of namelist not implemented yet"},
	{group: "w3c-ecma", name: "test355.txml"},
	{group: "w3c-ecma", name: "test364.txml", skip: "deep initial states are not supported"},
	{group: "w3c-ecma", name: "test372.txml"},
	{group: "w3c-ecma", name: "test375.txml"},
	{group: "w3c-ecma", name: "test376.txml", skip: "executable blocks not implemented"},
	{group: "w3c-ecma", name: "test377.txml"},
	{group: "w3c-ecma", name: "test378.txml", skip: "executable blocks not implemented"},
	{group: "w3c-ecma", name: "test387.txml"},
	{group: "w3c-ecma", name: "test388.txml", skip: "deep initial states are not supported"},
	{group: "w3c-ecma", name: "test396.txml"},
	{group: "w3c-ecma", name: "test399.txml"},
	{group: "w3c-ecma", name: "test401.txml", skip: "this assign to \"non-existent\" location in the datamodel, this is not exactly allowed in SCXML, but we don't disallow it - since u can assign to just any property on the `context` itself"},
	{group: "w3c-ecma", name: "test402.txml", skip: "TODO: investigate more, it expects error.execution when evaluating assign, check if assigning to a deep location is even allowed, check if assigning to an initialized datamodel is allowed, improve how datamodel is exposed to constructed functions"},
	{group: "w3c-ecma", name: "test403a.txml"},
	{group: "w3c-ecma", name: "test403b.txml"},
	{group: "w3c-ecma", name: "test403c.txml"},
	{group: "w3c-ecma", name: "test404.txml"},
	{group: "w3c-ecma", name: "test405.txml"},
	{group: "w3c-ecma", name: "test406.txml"},
	{group: "w3c-ecma", name: "test407.txml"},
	{group: "w3c-ecma", name: "test409.txml", skip: "conversion of In() predicate not implemented yet"},
	{group: "w3c-ecma", name: "test411.txml", skip: "conversion of In() predicate not implemented yet + microstep not implemented correctly"},
	{group: "w3c-ecma", name: "test412.txml", skip: "initial transitions with executable content not implemented yet"},
	{group: "w3c-ecma", name: "test413.txml", skip: "conversion of In() predicate not implemented yet"},
	{group: "w3c-ecma", name: "test416.txml"},
	{group: "w3c-ecma", name: "test417.txml"},
	{group: "w3c-ecma", name: "test419.txml"},
	{group: "w3c-ecma", name: "test421.txml"},
	{group: "w3c-ecma", name: "test422.txml", skip: "conversion of type-less <invoke> not implemented yet"},
	{group: "w3c-ecma", name: "test423.txml"},
	{group: "w3c-ecma", name: "test436.txml", skip: "conversion of In() predicate not implemented yet + null datamodel not implemented yet"},
	{group: "w3c-ecma", name: "test444.txml", skip: "datamodel being mutated in cond's expression 😱"},
	{group: "w3c-ecma", name: "test445.txml"},
	{group: "w3c-ecma", name: "test446.txml", skip: "conversion of <data src=\"...\"> not implemented yet"},
	{group: "w3c-ecma", name: "test448.txml", skip: "nested datamodels not implemented yet"},
	{group: "w3c-ecma", name: "test449.txml"},
	{group: "w3c-ecma", name: "test451.txml", skip: "conversion of In() predicate not implemented yet"},
	{group: "w3c-ecma", name: "test452.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test453.txml"},
	{group: "w3c-ecma", name: "test456.txml", skip: "conversion of <script> not implemented yet"},
	{group: "w3c-ecma", name: "test457.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test459.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test460.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test487.txml", skip: "this has a syntax error on purpose, so it's not included"},
	{group: "w3c-ecma", name: "test488.txml", skip: "error.execution when evaluating param"},
	{group: "w3c-ecma", name: "test495.txml"},
	{group: "w3c-ecma", name: "test496.txml", skip: "error.communication not implemented yet"},
	{group: "w3c-ecma", name: "test500.txml", skip: "_ioprocessors not yet available for expressions"},
	{group: "w3c-ecma", name: "test501.txml", skip: "_ioprocessors not yet available for expressions"},
	{group: "w3c-ecma", name: "test503.txml"},
	{group: "w3c-ecma", name: "test504.txml"},
	{group: "w3c-ecma", name: "test505.txml"},
	{group: "w3c-ecma", name: "test506.txml", skip: "`reenter` semantics in v5 are different from SCXML type=\"internal\"/\"external\" transitions, we respect `reenter` on all state types, not just on compound states"},
	{group: "w3c-ecma", name: "test509.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test510.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test518.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test519.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test520.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test521.txml", skip: "error.communication not implemented yet"},
	{group: "w3c-ecma", name: "test522.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test525.txml", skip: "<foreach> not implemented yet"},
	{group: "w3c-ecma", name: "test527.txml", skip: "conversion of <donedata> not implemented yet"},
	{group: "w3c-ecma", name: "test528.txml", skip: "conversion of <donedata> not implemented yet + error.execution when evaluating donedata"},
	{group: "w3c-ecma", name: "test529.txml", skip: "conversion of <donedata> not implemented yet"},
	{group: "w3c-ecma", name: "test530.txml", skip: "https://github.com/davidkpiano/xstate/pull/1811#discussion_r551897693"},
	{group: "w3c-ecma", name: "test531.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test532.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test533.txml", skip: "we allow `reenter: false` to not leave the source state even if that source state is not compound"},
	{group: "w3c-ecma", name: "test534.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test550.txml", skip: "non-root datamodel with early binding not implemented yet"},
	{group: "w3c-ecma", name: "test551.txml", skip: "non-root datamodel with early binding not implemented yet"},
	{group: "w3c-ecma", name: "test552.txml", skip: "conversion of <data src=\"...\"> not implemented yet"},
	{group: "w3c-ecma", name: "test553.txml", skip: "namelist not implemented yet + errored send not dispatching an event"},
	{group: "w3c-ecma", name: "test554.txml", skip: "namelist not implemented yet + errored invoke cancelled"},
	{group: "w3c-ecma", name: "test557.txml", skip: "conversion of <data src=\"...\"> not implemented yet"},
	{group: "w3c-ecma", name: "test558.txml", skip: "conversion of <data src=\"...\"> not implemented yet"},
	{group: "w3c-ecma", name: "test560.txml"},
	{group: "w3c-ecma", name: "test561.txml", skip: "processor creates an ECMAScript DOM object _event.data when receiving XML in an event"},
	{group: "w3c-ecma", name: "test562.txml", skip: "test that processor creates space normalized string in _event.data when receiving anything other than KVPs or XML in an event"},
	{group: "w3c-ecma", name: "test567.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test569.txml", skip: "_ioprocessors not yet available for expressions"},
	{group: "w3c-ecma", name: "test570.txml"},
	{group: "w3c-ecma", name: "test576.txml", skip: "multiple initial states are not supported"},
	{group: "w3c-ecma", name: "test577.txml", skip: "Basic HTTP Event I/O processor not implemented"},
	{group: "w3c-ecma", name: "test578.txml", skip: "conversion of <content> in <send> not implemented yet"},
	{group: "w3c-ecma", name: "test579.txml", skip: "executable content in history states not implemented yet"},
	{group: "w3c-ecma", name: "test580.txml", skip: "conversion of In() predicate not implemented yet"},
}

// scxmlSCIONTest mirrors the SCIONTest interface.
type scxmlSCIONTest struct {
	InitialConfiguration []string `json:"initialConfiguration"`
	Events               []struct {
		After float64 `json:"after"`
		Event struct {
			Name string `json:"name"`
		} `json:"event"`
		NextConfiguration []string `json:"nextConfiguration"`
	} `json:"events"`
}

// scxmlW3TestTimeout stands in for the default vitest test timeout that
// bounds `await runW3TestToCompletion(...)` in JS.
const scxmlW3TestTimeout = 5 * time.Second

// scxmlValueJSON mirrors JSON.stringify(prevState?.value).
func scxmlValueJSON(s scxmlSnapshot) string {
	if s == nil {
		return "undefined"
	}
	b, err := json.Marshal(s.Value)
	if err != nil {
		return fmt.Sprint(s.Value)
	}
	return string(b)
}

// scxmlRunW3TestToCompletion mirrors runW3TestToCompletion: the machine must
// complete in the "final" or "pass" state. Real timers are used, as in JS.
func scxmlRunW3TestToCompletion(t *testing.T, machine scxmlMachine) {
	t.Helper()
	var mu sync.Mutex
	var prevState, nextState scxmlSnapshot
	result := make(chan error, 1)

	actor := xs.CreateActor(machine, xs.WithLogger(func(...any) {}))
	actor.Subscribe(xs.Observer[scxmlSnapshot]{
		Next: func(s scxmlSnapshot) {
			mu.Lock()
			defer mu.Unlock()
			prevState = nextState
			nextState = s
		},
		Complete: func() {
			mu.Lock()
			defer mu.Unlock()
			if nextState == nil {
				result <- fmt.Errorf("completed before emitting a snapshot")
				return
			}
			// Add 'final' for test230.txml which does not have a 'pass' state
			if v, _ := nextState.Value.(string); v == "final" || v == "pass" {
				result <- nil
				return
			}
			result <- fmt.Errorf("Reached \"fail\" state from state %s", scxmlValueJSON(prevState))
		},
	})
	actor.Start()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(scxmlW3TestTimeout):
		t.Fatalf("machine did not complete within %s", scxmlW3TestTimeout)
	}
}

// scxmlRunTestToCompletion mirrors runTestToCompletion: it replays the
// events with a simulated clock and checks the first expected state of every
// next configuration is active.
func scxmlRunTestToCompletion(t *testing.T, machine scxmlMachine, test scxmlSCIONTest) {
	t.Helper()
	if len(test.Events) == 0 && len(test.InitialConfiguration) > 0 && test.InitialConfiguration[0] == "pass" {
		scxmlRunW3TestToCompletion(t, machine)
		return
	}

	var mu sync.Mutex
	done := false
	clock := xs.NewSimulatedClock()
	service := xs.CreateActor(machine, xs.WithClock(clock))

	nextState := service.GetSnapshot()
	var prevState scxmlSnapshot
	service.SubscribeNext(func(s scxmlSnapshot) {
		mu.Lock()
		defer mu.Unlock()
		prevState = nextState
		nextState = s
	})
	service.Subscribe(xs.Observer[scxmlSnapshot]{
		Complete: func() {
			mu.Lock()
			defer mu.Unlock()
			if v, _ := nextState.Value.(string); v == "fail" {
				// JS throws from the complete observer; done stays false.
				t.Errorf("Reached \"fail\" state from state %s", scxmlValueJSON(prevState))
				return
			}
			done = true
		},
	})
	service.Start()

	for _, ev := range test.Events {
		mu.Lock()
		isDone := done
		mu.Unlock()
		if isDone {
			// JS returns from every remaining forEach callback once done.
			break
		}
		if ev.After != 0 {
			clock.Increment(time.Duration(ev.After * float64(time.Millisecond)))
		}
		service.Send(xs.Ev(ev.Event.Name))

		mu.Lock()
		value := nextState.Value
		mu.Unlock()
		var stateIDs []string
		for _, n := range xs.GetStateNodes(machine.Root, value) {
			stateIDs = append(stateIDs, n.ID)
		}

		assert.Contains(t, stateIDs, scxml.SanitizeStateID(ev.NextConfiguration[0]))
	}
}

// JS: scxml > ${testGroupName}/${testName} (one subtest per entry of scxmlCases)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/scxml.test.ts#L472
func TestSCXML(t *testing.T) {
	for _, tc := range scxmlCases {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/scxml.test.ts#L472
		t.Run(tc.group+"/"+tc.name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip("skipped in JS: " + tc.skip)
			}
			dir := filepath.Join("testdata", tc.group)
			scxmlDefinition, err := os.ReadFile(filepath.Join(dir, tc.name+".scxml"))
			require.NoError(t, err)
			raw, err := os.ReadFile(filepath.Join(dir, tc.name+".json"))
			require.NoError(t, err)
			var scxmlTest scxmlSCIONTest
			require.NoError(t, json.Unmarshal(raw, &scxmlTest))

			machine := scxml.ToMachine(string(scxmlDefinition))

			scxmlRunTestToCompletion(t, machine, scxmlTest)
		})
	}
}
