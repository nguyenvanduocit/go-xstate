> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# Port manifest: scxml

Source: `references/xstate/packages/core/test/scxml.test.ts` (whole file). Go: `scxml/scxml_test.go` (`//go:build port_scxml`), table-driven `TestSCXML`, one subtest per `(testGroupName, testName)` entry of the JS `testGroups`, commented-out entries included as skipped subtests carrying the JS comment.

Totals: 323 cases = 169 `ported` + 154 `skipped-in-JS` (commented out in `testGroups`). No `N/A-type` / `N/A-runtime`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| scxml > actionSend/send1 | `TestSCXML/actionSend/send1` | ported |  |
| scxml > actionSend/send2 | `TestSCXML/actionSend/send2` | ported |  |
| scxml > actionSend/send3 | `TestSCXML/actionSend/send3` | ported |  |
| scxml > actionSend/send4 | `TestSCXML/actionSend/send4` | ported |  |
| scxml > actionSend/send4b | `TestSCXML/actionSend/send4b` | ported |  |
| scxml > actionSend/send7 | `TestSCXML/actionSend/send7` | ported |  |
| scxml > actionSend/send7b | `TestSCXML/actionSend/send7b` | ported |  |
| scxml > actionSend/send8 | `TestSCXML/actionSend/send8` | ported |  |
| scxml > actionSend/send8b | `TestSCXML/actionSend/send8b` | ported |  |
| scxml > actionSend/send9 | `TestSCXML/actionSend/send9` | ported |  |
| scxml > assign/assign_invalid | `TestSCXML/assign/assign_invalid` | skipped-in-JS | commented out in JS: this has a syntax error on purpose, so it's not included |
| scxml > assign/assign_obj_literal | `TestSCXML/assign/assign_obj_literal` | skipped-in-JS | commented out in JS: deep initial states are not supported |
| scxml > assign-current-small-step/test0 | `TestSCXML/assign-current-small-step/test0` | ported | `.scxml` from xstate `test/fixtures/scxml` (JS `overrides`); `.json` from scion |
| scxml > assign-current-small-step/test1 | `TestSCXML/assign-current-small-step/test1` | ported |  |
| scxml > assign-current-small-step/test2 | `TestSCXML/assign-current-small-step/test2` | ported |  |
| scxml > assign-current-small-step/test3 | `TestSCXML/assign-current-small-step/test3` | ported |  |
| scxml > assign-current-small-step/test4 | `TestSCXML/assign-current-small-step/test4` | ported |  |
| scxml > basic/basic0 | `TestSCXML/basic/basic0` | ported |  |
| scxml > basic/basic1 | `TestSCXML/basic/basic1` | ported |  |
| scxml > basic/basic2 | `TestSCXML/basic/basic2` | ported |  |
| scxml > cond-js/test0 | `TestSCXML/cond-js/test0` | ported |  |
| scxml > cond-js/test1 | `TestSCXML/cond-js/test1` | ported |  |
| scxml > cond-js/test2 | `TestSCXML/cond-js/test2` | ported |  |
| scxml > cond-js/TestConditionalTransition | `TestSCXML/cond-js/TestConditionalTransition` | ported |  |
| scxml > data/data_invalid | `TestSCXML/data/data_invalid` | skipped-in-JS | commented out in JS: commented out in JS without a reason |
| scxml > data/data_obj_literal | `TestSCXML/data/data_obj_literal` | skipped-in-JS | commented out in JS: deep initial states are not supported |
| scxml > default-initial-state/initial1 | `TestSCXML/default-initial-state/initial1` | ported |  |
| scxml > default-initial-state/initial2 | `TestSCXML/default-initial-state/initial2` | ported |  |
| scxml > delayedSend/send1 | `TestSCXML/delayedSend/send1` | ported |  |
| scxml > delayedSend/send2 | `TestSCXML/delayedSend/send2` | ported |  |
| scxml > delayedSend/send3 | `TestSCXML/delayedSend/send3` | ported |  |
| scxml > documentOrder/documentOrder0 | `TestSCXML/documentOrder/documentOrder0` | ported |  |
| scxml > error/error | `TestSCXML/error/error` | skipped-in-JS | commented out in JS: not implemented |
| scxml > forEach/test1 | `TestSCXML/forEach/test1` | skipped-in-JS | commented out in JS: not implemented |
| scxml > hierarchy/hier0 | `TestSCXML/hierarchy/hier0` | ported |  |
| scxml > hierarchy/hier1 | `TestSCXML/hierarchy/hier1` | ported |  |
| scxml > hierarchy/hier2 | `TestSCXML/hierarchy/hier2` | ported |  |
| scxml > hierarchy+documentOrder/test0 | `TestSCXML/hierarchy+documentOrder/test0` | ported |  |
| scxml > hierarchy+documentOrder/test1 | `TestSCXML/hierarchy+documentOrder/test1` | ported |  |
| scxml > history/history0 | `TestSCXML/history/history0` | ported |  |
| scxml > history/history1 | `TestSCXML/history/history1` | ported |  |
| scxml > history/history2 | `TestSCXML/history/history2` | ported |  |
| scxml > history/history3 | `TestSCXML/history/history3` | ported |  |
| scxml > history/history4 | `TestSCXML/history/history4` | ported |  |
| scxml > history/history4b | `TestSCXML/history/history4b` | ported |  |
| scxml > history/history5 | `TestSCXML/history/history5` | ported |  |
| scxml > history/history6 | `TestSCXML/history/history6` | ported |  |
| scxml > if-else/test0 | `TestSCXML/if-else/test0` | ported |  |
| scxml > in/TestInPredicate | `TestSCXML/in/TestInPredicate` | ported |  |
| scxml > internal-transitions/test0 | `TestSCXML/internal-transitions/test0` | ported |  |
| scxml > internal-transitions/test1 | `TestSCXML/internal-transitions/test1` | ported |  |
| scxml > misc/deep-initial | `TestSCXML/misc/deep-initial` | skipped-in-JS | commented out in JS: deep initial states are not supported |
| scxml > more-parallel/test0 | `TestSCXML/more-parallel/test0` | ported |  |
| scxml > more-parallel/test1 | `TestSCXML/more-parallel/test1` | ported |  |
| scxml > more-parallel/test2 | `TestSCXML/more-parallel/test2` | ported |  |
| scxml > more-parallel/test2b | `TestSCXML/more-parallel/test2b` | ported |  |
| scxml > more-parallel/test3 | `TestSCXML/more-parallel/test3` | ported |  |
| scxml > more-parallel/test3b | `TestSCXML/more-parallel/test3b` | ported |  |
| scxml > more-parallel/test4 | `TestSCXML/more-parallel/test4` | ported |  |
| scxml > more-parallel/test5 | `TestSCXML/more-parallel/test5` | ported |  |
| scxml > more-parallel/test6 | `TestSCXML/more-parallel/test6` | ported |  |
| scxml > more-parallel/test6b | `TestSCXML/more-parallel/test6b` | ported |  |
| scxml > more-parallel/test7 | `TestSCXML/more-parallel/test7` | ported |  |
| scxml > more-parallel/test8 | `TestSCXML/more-parallel/test8` | ported |  |
| scxml > more-parallel/test9 | `TestSCXML/more-parallel/test9` | ported |  |
| scxml > more-parallel/test10 | `TestSCXML/more-parallel/test10` | ported |  |
| scxml > more-parallel/test10b | `TestSCXML/more-parallel/test10b` | ported |  |
| scxml > multiple-events-per-transition/test1 | `TestSCXML/multiple-events-per-transition/test1` | ported |  |
| scxml > parallel/test0 | `TestSCXML/parallel/test0` | ported |  |
| scxml > parallel/test1 | `TestSCXML/parallel/test1` | ported |  |
| scxml > parallel/test2 | `TestSCXML/parallel/test2` | ported |  |
| scxml > parallel/test3 | `TestSCXML/parallel/test3` | ported |  |
| scxml > parallel+interrupt/test0 | `TestSCXML/parallel+interrupt/test0` | ported |  |
| scxml > parallel+interrupt/test1 | `TestSCXML/parallel+interrupt/test1` | ported |  |
| scxml > parallel+interrupt/test2 | `TestSCXML/parallel+interrupt/test2` | ported |  |
| scxml > parallel+interrupt/test3 | `TestSCXML/parallel+interrupt/test3` | ported |  |
| scxml > parallel+interrupt/test4 | `TestSCXML/parallel+interrupt/test4` | ported |  |
| scxml > parallel+interrupt/test5 | `TestSCXML/parallel+interrupt/test5` | ported |  |
| scxml > parallel+interrupt/test6 | `TestSCXML/parallel+interrupt/test6` | ported |  |
| scxml > parallel+interrupt/test7 | `TestSCXML/parallel+interrupt/test7` | ported |  |
| scxml > parallel+interrupt/test7b | `TestSCXML/parallel+interrupt/test7b` | ported |  |
| scxml > parallel+interrupt/test8 | `TestSCXML/parallel+interrupt/test8` | ported |  |
| scxml > parallel+interrupt/test9 | `TestSCXML/parallel+interrupt/test9` | ported |  |
| scxml > parallel+interrupt/test10 | `TestSCXML/parallel+interrupt/test10` | ported |  |
| scxml > parallel+interrupt/test11 | `TestSCXML/parallel+interrupt/test11` | ported |  |
| scxml > parallel+interrupt/test12 | `TestSCXML/parallel+interrupt/test12` | ported |  |
| scxml > parallel+interrupt/test13 | `TestSCXML/parallel+interrupt/test13` | ported |  |
| scxml > parallel+interrupt/test14 | `TestSCXML/parallel+interrupt/test14` | ported |  |
| scxml > parallel+interrupt/test15 | `TestSCXML/parallel+interrupt/test15` | ported |  |
| scxml > parallel+interrupt/test16 | `TestSCXML/parallel+interrupt/test16` | ported |  |
| scxml > parallel+interrupt/test17 | `TestSCXML/parallel+interrupt/test17` | ported |  |
| scxml > parallel+interrupt/test18 | `TestSCXML/parallel+interrupt/test18` | ported |  |
| scxml > parallel+interrupt/test19 | `TestSCXML/parallel+interrupt/test19` | ported |  |
| scxml > parallel+interrupt/test20 | `TestSCXML/parallel+interrupt/test20` | ported |  |
| scxml > parallel+interrupt/test21 | `TestSCXML/parallel+interrupt/test21` | ported |  |
| scxml > parallel+interrupt/test21b | `TestSCXML/parallel+interrupt/test21b` | ported |  |
| scxml > parallel+interrupt/test21c | `TestSCXML/parallel+interrupt/test21c` | ported |  |
| scxml > parallel+interrupt/test22 | `TestSCXML/parallel+interrupt/test22` | ported |  |
| scxml > parallel+interrupt/test23 | `TestSCXML/parallel+interrupt/test23` | ported |  |
| scxml > parallel+interrupt/test24 | `TestSCXML/parallel+interrupt/test24` | ported |  |
| scxml > parallel+interrupt/test25 | `TestSCXML/parallel+interrupt/test25` | ported |  |
| scxml > parallel+interrupt/test27 | `TestSCXML/parallel+interrupt/test27` | ported |  |
| scxml > parallel+interrupt/test28 | `TestSCXML/parallel+interrupt/test28` | ported |  |
| scxml > parallel+interrupt/test29 | `TestSCXML/parallel+interrupt/test29` | ported |  |
| scxml > parallel+interrupt/test30 | `TestSCXML/parallel+interrupt/test30` | ported |  |
| scxml > parallel+interrupt/test31 | `TestSCXML/parallel+interrupt/test31` | ported |  |
| scxml > script/test0 | `TestSCXML/script/test0` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script/test1 | `TestSCXML/script/test1` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script/test2 | `TestSCXML/script/test2` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script-src/test0 | `TestSCXML/script-src/test0` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script-src/test1 | `TestSCXML/script-src/test1` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script-src/test2 | `TestSCXML/script-src/test2` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > script-src/test3 | `TestSCXML/script-src/test3` | skipped-in-JS | commented out in JS: <script/> conversion not implemented |
| scxml > scxml-prefix-event-name-matching/star0 | `TestSCXML/scxml-prefix-event-name-matching/star0` | skipped-in-JS | commented out in JS: this relies on the source order of transitions where * is first and it's supposed to get matched over an explicit descriptor |
| scxml > scxml-prefix-event-name-matching/test0 | `TestSCXML/scxml-prefix-event-name-matching/test0` | skipped-in-JS | commented out in JS: prefix event matching not implemented yet |
| scxml > scxml-prefix-event-name-matching/test1 | `TestSCXML/scxml-prefix-event-name-matching/test1` | skipped-in-JS | commented out in JS: prefix event matching not implemented yet |
| scxml > send-data/send1 | `TestSCXML/send-data/send1` | skipped-in-JS | commented out in JS: <content> conversion not implemented |
| scxml > send-idlocation/test0 | `TestSCXML/send-idlocation/test0` | skipped-in-JS | commented out in JS: commented out in JS without a reason |
| scxml > send-internal/test0 | `TestSCXML/send-internal/test0` | skipped-in-JS | commented out in JS: commented out in JS without a reason |
| scxml > targetless-transition/test0 | `TestSCXML/targetless-transition/test0` | ported |  |
| scxml > targetless-transition/test1 | `TestSCXML/targetless-transition/test1` | ported |  |
| scxml > targetless-transition/test2 | `TestSCXML/targetless-transition/test2` | ported |  |
| scxml > targetless-transition/test3 | `TestSCXML/targetless-transition/test3` | ported |  |
| scxml > w3c-ecma/test144.txml | `TestSCXML/w3c-ecma/test144.txml` | ported |  |
| scxml > w3c-ecma/test147.txml | `TestSCXML/w3c-ecma/test147.txml` | ported |  |
| scxml > w3c-ecma/test148.txml | `TestSCXML/w3c-ecma/test148.txml` | ported |  |
| scxml > w3c-ecma/test149.txml | `TestSCXML/w3c-ecma/test149.txml` | ported |  |
| scxml > w3c-ecma/test150.txml | `TestSCXML/w3c-ecma/test150.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test151.txml | `TestSCXML/w3c-ecma/test151.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test152.txml | `TestSCXML/w3c-ecma/test152.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test153.txml | `TestSCXML/w3c-ecma/test153.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test155.txml | `TestSCXML/w3c-ecma/test155.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test156.txml | `TestSCXML/w3c-ecma/test156.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test158.txml | `TestSCXML/w3c-ecma/test158.txml` | ported |  |
| scxml > w3c-ecma/test159.txml | `TestSCXML/w3c-ecma/test159.txml` | skipped-in-JS | commented out in JS: different error handling |
| scxml > w3c-ecma/test172.txml | `TestSCXML/w3c-ecma/test172.txml` | ported |  |
| scxml > w3c-ecma/test173.txml | `TestSCXML/w3c-ecma/test173.txml` | ported |  |
| scxml > w3c-ecma/test174.txml | `TestSCXML/w3c-ecma/test174.txml` | ported |  |
| scxml > w3c-ecma/test175.txml | `TestSCXML/w3c-ecma/test175.txml` | ported |  |
| scxml > w3c-ecma/test176.txml | `TestSCXML/w3c-ecma/test176.txml` | ported |  |
| scxml > w3c-ecma/test179.txml | `TestSCXML/w3c-ecma/test179.txml` | skipped-in-JS | commented out in JS: conversion of <content> in <send> not implemented yet |
| scxml > w3c-ecma/test183.txml | `TestSCXML/w3c-ecma/test183.txml` | skipped-in-JS | commented out in JS: idlocation not implemented yet |
| scxml > w3c-ecma/test185.txml | `TestSCXML/w3c-ecma/test185.txml` | ported |  |
| scxml > w3c-ecma/test186.txml | `TestSCXML/w3c-ecma/test186.txml` | ported |  |
| scxml > w3c-ecma/test187.txml | `TestSCXML/w3c-ecma/test187.txml` | ported |  |
| scxml > w3c-ecma/test189.txml | `TestSCXML/w3c-ecma/test189.txml` | ported |  |
| scxml > w3c-ecma/test190.txml | `TestSCXML/w3c-ecma/test190.txml` | ported | JS note: _sessionid is undefined for expressions |
| scxml > w3c-ecma/test191.txml | `TestSCXML/w3c-ecma/test191.txml` | ported |  |
| scxml > w3c-ecma/test192.txml | `TestSCXML/w3c-ecma/test192.txml` | ported |  |
| scxml > w3c-ecma/test193.txml | `TestSCXML/w3c-ecma/test193.txml` | ported |  |
| scxml > w3c-ecma/test194.txml | `TestSCXML/w3c-ecma/test194.txml` | skipped-in-JS | commented out in JS: it's using an invalid event target (another actor), we should be erroring on this somehow when we revamp the error story |
| scxml > w3c-ecma/test198.txml | `TestSCXML/w3c-ecma/test198.txml` | skipped-in-JS | commented out in JS: origintype not implemented yet |
| scxml > w3c-ecma/test199.txml | `TestSCXML/w3c-ecma/test199.txml` | skipped-in-JS | commented out in JS: send type not checked |
| scxml > w3c-ecma/test200.txml | `TestSCXML/w3c-ecma/test200.txml` | ported |  |
| scxml > w3c-ecma/test201.txml | `TestSCXML/w3c-ecma/test201.txml` | ported |  |
| scxml > w3c-ecma/test205.txml | `TestSCXML/w3c-ecma/test205.txml` | ported |  |
| scxml > w3c-ecma/test207.txml | `TestSCXML/w3c-ecma/test207.txml` | skipped-in-JS | commented out in JS: delayexpr |
| scxml > w3c-ecma/test208.txml | `TestSCXML/w3c-ecma/test208.txml` | ported |  |
| scxml > w3c-ecma/test210.txml | `TestSCXML/w3c-ecma/test210.txml` | skipped-in-JS | commented out in JS: sendidexpr not supported yet |
| scxml > w3c-ecma/test215.txml | `TestSCXML/w3c-ecma/test215.txml` | skipped-in-JS | commented out in JS: <invoke typeexpr="..."> |
| scxml > w3c-ecma/test216.txml | `TestSCXML/w3c-ecma/test216.txml` | skipped-in-JS | commented out in JS: <invoke srcexpr="..."> |
| scxml > w3c-ecma/test220.txml | `TestSCXML/w3c-ecma/test220.txml` | ported |  |
| scxml > w3c-ecma/test223.txml | `TestSCXML/w3c-ecma/test223.txml` | skipped-in-JS | commented out in JS: idlocation not implemented yet |
| scxml > w3c-ecma/test224.txml | `TestSCXML/w3c-ecma/test224.txml` | skipped-in-JS | commented out in JS: <invoke idlocation="..."> |
| scxml > w3c-ecma/test225.txml | `TestSCXML/w3c-ecma/test225.txml` | skipped-in-JS | commented out in JS: unique invokeids generated at invoke time |
| scxml > w3c-ecma/test226.txml | `TestSCXML/w3c-ecma/test226.txml` | skipped-in-JS | commented out in JS: <invoke src="..."> |
| scxml > w3c-ecma/test228.txml | `TestSCXML/w3c-ecma/test228.txml` | skipped-in-JS | commented out in JS: this test relies on `invokeid` being available on the event |
| scxml > w3c-ecma/test229.txml | `TestSCXML/w3c-ecma/test229.txml` | skipped-in-JS | commented out in JS: autoForward not supported in v5 |
| scxml > w3c-ecma/test230.txml | `TestSCXML/w3c-ecma/test230.txml` | skipped-in-JS | commented out in JS: autoForward not supported in v5 |
| scxml > w3c-ecma/test232.txml | `TestSCXML/w3c-ecma/test232.txml` | ported |  |
| scxml > w3c-ecma/test233.txml | `TestSCXML/w3c-ecma/test233.txml` | skipped-in-JS | commented out in JS: <finalize> not implemented yet |
| scxml > w3c-ecma/test234.txml | `TestSCXML/w3c-ecma/test234.txml` | skipped-in-JS | commented out in JS: <finalize> not implemented yet |
| scxml > w3c-ecma/test235.txml | `TestSCXML/w3c-ecma/test235.txml` | ported |  |
| scxml > w3c-ecma/test236.txml | `TestSCXML/w3c-ecma/test236.txml` | skipped-in-JS | commented out in JS: reaching a final state should execute all onexit handlers |
| scxml > w3c-ecma/test237.txml | `TestSCXML/w3c-ecma/test237.txml` | ported |  |
| scxml > w3c-ecma/test239.txml | `TestSCXML/w3c-ecma/test239.txml` | skipped-in-JS | commented out in JS: <invoke src="..."> |
| scxml > w3c-ecma/test240.txml | `TestSCXML/w3c-ecma/test240.txml` | skipped-in-JS | commented out in JS: conversion of namelist not implemented yet |
| scxml > w3c-ecma/test241.txml | `TestSCXML/w3c-ecma/test241.txml` | skipped-in-JS | commented out in JS: conversion of namelist not implemented yet |
| scxml > w3c-ecma/test242.txml | `TestSCXML/w3c-ecma/test242.txml` | skipped-in-JS | commented out in JS: <invoke src="..."> |
| scxml > w3c-ecma/test243.txml | `TestSCXML/w3c-ecma/test243.txml` | skipped-in-JS | commented out in JS: conversion of <param> in <scxml> not implemented yet |
| scxml > w3c-ecma/test244.txml | `TestSCXML/w3c-ecma/test244.txml` | skipped-in-JS | commented out in JS: conversion of namelist not implemented yet |
| scxml > w3c-ecma/test245.txml | `TestSCXML/w3c-ecma/test245.txml` | skipped-in-JS | commented out in JS: conversion of namelist not implemented yet |
| scxml > w3c-ecma/test247.txml | `TestSCXML/w3c-ecma/test247.txml` | ported |  |
| scxml > w3c-ecma/test250.txml | `TestSCXML/w3c-ecma/test250.txml` | skipped-in-JS | commented out in JS: this is a manual test - we could test it by snapshotting logged valued |
| scxml > w3c-ecma/test252.txml | `TestSCXML/w3c-ecma/test252.txml` | skipped-in-JS | commented out in JS: this expects the parent to not receive the event sent from the canceled child's exit action |
| scxml > w3c-ecma/test253.txml | `TestSCXML/w3c-ecma/test253.txml` | skipped-in-JS | commented out in JS: _event.origintype not implemented yet |
| scxml > w3c-ecma/test276.txml | `TestSCXML/w3c-ecma/test276.txml` | skipped-in-JS | commented out in JS: <invoke src="..."> |
| scxml > w3c-ecma/test277.txml | `TestSCXML/w3c-ecma/test277.txml` | skipped-in-JS | commented out in JS: illegal expression in datamodel creates unbound variable |
| scxml > w3c-ecma/test278.txml | `TestSCXML/w3c-ecma/test278.txml` | skipped-in-JS | commented out in JS: non-root datamodel with early binding not implemented yet |
| scxml > w3c-ecma/test279.txml | `TestSCXML/w3c-ecma/test279.txml` | skipped-in-JS | commented out in JS: non-root datamodel with early binding not implemented yet |
| scxml > w3c-ecma/test280.txml | `TestSCXML/w3c-ecma/test280.txml` | skipped-in-JS | commented out in JS: non-root datamodel with late binding not implemented yet |
| scxml > w3c-ecma/test286.txml | `TestSCXML/w3c-ecma/test286.txml` | skipped-in-JS | commented out in JS: this intentionally throws when executing assign, we should be erroring on this somehow when we revamp the error story |
| scxml > w3c-ecma/test287.txml | `TestSCXML/w3c-ecma/test287.txml` | ported |  |
| scxml > w3c-ecma/test294.txml | `TestSCXML/w3c-ecma/test294.txml` | skipped-in-JS | commented out in JS: conversion of <donedata> not implemented yet |
| scxml > w3c-ecma/test298.txml | `TestSCXML/w3c-ecma/test298.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating donedata |
| scxml > w3c-ecma/test302.txml | `TestSCXML/w3c-ecma/test302.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test303-1.txml | `TestSCXML/w3c-ecma/test303-1.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test303-2.txml | `TestSCXML/w3c-ecma/test303-2.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test303.txml | `TestSCXML/w3c-ecma/test303.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test304.txml | `TestSCXML/w3c-ecma/test304.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test307.txml | `TestSCXML/w3c-ecma/test307.txml` | skipped-in-JS | commented out in JS: non-root datamodel with late binding not implemented yet |
| scxml > w3c-ecma/test309.txml | `TestSCXML/w3c-ecma/test309.txml` | skipped-in-JS | commented out in JS: error in cond expression being treated as false |
| scxml > w3c-ecma/test310.txml | `TestSCXML/w3c-ecma/test310.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet |
| scxml > w3c-ecma/test311.txml | `TestSCXML/w3c-ecma/test311.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating assign |
| scxml > w3c-ecma/test312.txml | `TestSCXML/w3c-ecma/test312.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating assign |
| scxml > w3c-ecma/test313.txml | `TestSCXML/w3c-ecma/test313.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating assign |
| scxml > w3c-ecma/test314.txml | `TestSCXML/w3c-ecma/test314.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating assign |
| scxml > w3c-ecma/test318.txml | `TestSCXML/w3c-ecma/test318.txml` | ported |  |
| scxml > w3c-ecma/test319.txml | `TestSCXML/w3c-ecma/test319.txml` | skipped-in-JS | commented out in JS: SCXML has no init event, so _event stays unbound in onentry of initial state |
| scxml > w3c-ecma/test321.txml | `TestSCXML/w3c-ecma/test321.txml` | skipped-in-JS | commented out in JS: _sessionid not yet available for expressions |
| scxml > w3c-ecma/test322.txml | `TestSCXML/w3c-ecma/test322.txml` | skipped-in-JS | commented out in JS: _sessionid not yet available for expressions |
| scxml > w3c-ecma/test323.txml | `TestSCXML/w3c-ecma/test323.txml` | skipped-in-JS | commented out in JS: _name not yet available for expressions |
| scxml > w3c-ecma/test324.txml | `TestSCXML/w3c-ecma/test324.txml` | skipped-in-JS | commented out in JS: _name not yet available for expressions |
| scxml > w3c-ecma/test325.txml | `TestSCXML/w3c-ecma/test325.txml` | skipped-in-JS | commented out in JS: _ioprocessors not yet available for expressions |
| scxml > w3c-ecma/test326.txml | `TestSCXML/w3c-ecma/test326.txml` | skipped-in-JS | commented out in JS: _ioprocessors not yet available for expressions |
| scxml > w3c-ecma/test329.txml | `TestSCXML/w3c-ecma/test329.txml` | skipped-in-JS | commented out in JS: system variables can't be modified, we don't keep them in datamodel, so it might be hard to run this test |
| scxml > w3c-ecma/test330.txml | `TestSCXML/w3c-ecma/test330.txml` | skipped-in-JS | commented out in JS: SCXML _event properties not implemented yet |
| scxml > w3c-ecma/test331.txml | `TestSCXML/w3c-ecma/test331.txml` | skipped-in-JS | commented out in JS: _event.type not implemented yet correctly |
| scxml > w3c-ecma/test332.txml | `TestSCXML/w3c-ecma/test332.txml` | skipped-in-JS | commented out in JS: idlocation not implemented yet |
| scxml > w3c-ecma/test333.txml | `TestSCXML/w3c-ecma/test333.txml` | ported |  |
| scxml > w3c-ecma/test335.txml | `TestSCXML/w3c-ecma/test335.txml` | ported |  |
| scxml > w3c-ecma/test336.txml | `TestSCXML/w3c-ecma/test336.txml` | ported |  |
| scxml > w3c-ecma/test337.txml | `TestSCXML/w3c-ecma/test337.txml` | ported |  |
| scxml > w3c-ecma/test338.txml | `TestSCXML/w3c-ecma/test338.txml` | skipped-in-JS | commented out in JS: <invoke idlocation="..."> + _event.invokeid available on <send> events received from the invoked child |
| scxml > w3c-ecma/test339.txml | `TestSCXML/w3c-ecma/test339.txml` | ported |  |
| scxml > w3c-ecma/test342.txml | `TestSCXML/w3c-ecma/test342.txml` | ported |  |
| scxml > w3c-ecma/test343.txml | `TestSCXML/w3c-ecma/test343.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating donedata |
| scxml > w3c-ecma/test344.txml | `TestSCXML/w3c-ecma/test344.txml` | skipped-in-JS | commented out in JS: error in cond expression being treated as false and raises error.execution |
| scxml > w3c-ecma/test346.txml | `TestSCXML/w3c-ecma/test346.txml` | skipped-in-JS | commented out in JS: system variables can't be modified, we don't keep them in datamodel, so it might be hard to run this test |
| scxml > w3c-ecma/test347.txml | `TestSCXML/w3c-ecma/test347.txml` | ported |  |
| scxml > w3c-ecma/test348.txml | `TestSCXML/w3c-ecma/test348.txml` | ported |  |
| scxml > w3c-ecma/test349.txml | `TestSCXML/w3c-ecma/test349.txml` | ported |  |
| scxml > w3c-ecma/test350.txml | `TestSCXML/w3c-ecma/test350.txml` | skipped-in-JS | commented out in JS: _sessionid not yet available for expressions |
| scxml > w3c-ecma/test351.txml | `TestSCXML/w3c-ecma/test351.txml` | skipped-in-JS | commented out in JS: _event.sendid not implemented yet |
| scxml > w3c-ecma/test352.txml | `TestSCXML/w3c-ecma/test352.txml` | skipped-in-JS | commented out in JS: _event.origintype not implemented yet |
| scxml > w3c-ecma/test354.txml | `TestSCXML/w3c-ecma/test354.txml` | skipped-in-JS | commented out in JS: conversion of namelist not implemented yet |
| scxml > w3c-ecma/test355.txml | `TestSCXML/w3c-ecma/test355.txml` | ported |  |
| scxml > w3c-ecma/test364.txml | `TestSCXML/w3c-ecma/test364.txml` | skipped-in-JS | commented out in JS: deep initial states are not supported |
| scxml > w3c-ecma/test372.txml | `TestSCXML/w3c-ecma/test372.txml` | ported |  |
| scxml > w3c-ecma/test375.txml | `TestSCXML/w3c-ecma/test375.txml` | ported |  |
| scxml > w3c-ecma/test376.txml | `TestSCXML/w3c-ecma/test376.txml` | skipped-in-JS | commented out in JS: executable blocks not implemented |
| scxml > w3c-ecma/test377.txml | `TestSCXML/w3c-ecma/test377.txml` | ported |  |
| scxml > w3c-ecma/test378.txml | `TestSCXML/w3c-ecma/test378.txml` | skipped-in-JS | commented out in JS: executable blocks not implemented |
| scxml > w3c-ecma/test387.txml | `TestSCXML/w3c-ecma/test387.txml` | ported |  |
| scxml > w3c-ecma/test388.txml | `TestSCXML/w3c-ecma/test388.txml` | skipped-in-JS | commented out in JS: deep initial states are not supported |
| scxml > w3c-ecma/test396.txml | `TestSCXML/w3c-ecma/test396.txml` | ported |  |
| scxml > w3c-ecma/test399.txml | `TestSCXML/w3c-ecma/test399.txml` | ported |  |
| scxml > w3c-ecma/test401.txml | `TestSCXML/w3c-ecma/test401.txml` | skipped-in-JS | commented out in JS: this assign to "non-existent" location in the datamodel, this is not exactly allowed in SCXML, but we don't disallow it - since u can assign to just any property on the `context` itself |
| scxml > w3c-ecma/test402.txml | `TestSCXML/w3c-ecma/test402.txml` | skipped-in-JS | commented out in JS: TODO: investigate more, it expects error.execution when evaluating assign, check if assigning to a deep location is even allowed, check if assigning to an initialized datamodel is allowed, improve how datamodel is exposed to constructed functions |
| scxml > w3c-ecma/test403a.txml | `TestSCXML/w3c-ecma/test403a.txml` | ported |  |
| scxml > w3c-ecma/test403b.txml | `TestSCXML/w3c-ecma/test403b.txml` | ported |  |
| scxml > w3c-ecma/test403c.txml | `TestSCXML/w3c-ecma/test403c.txml` | ported |  |
| scxml > w3c-ecma/test404.txml | `TestSCXML/w3c-ecma/test404.txml` | ported |  |
| scxml > w3c-ecma/test405.txml | `TestSCXML/w3c-ecma/test405.txml` | ported |  |
| scxml > w3c-ecma/test406.txml | `TestSCXML/w3c-ecma/test406.txml` | ported |  |
| scxml > w3c-ecma/test407.txml | `TestSCXML/w3c-ecma/test407.txml` | ported |  |
| scxml > w3c-ecma/test409.txml | `TestSCXML/w3c-ecma/test409.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet |
| scxml > w3c-ecma/test411.txml | `TestSCXML/w3c-ecma/test411.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet + microstep not implemented correctly |
| scxml > w3c-ecma/test412.txml | `TestSCXML/w3c-ecma/test412.txml` | skipped-in-JS | commented out in JS: initial transitions with executable content not implemented yet |
| scxml > w3c-ecma/test413.txml | `TestSCXML/w3c-ecma/test413.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet |
| scxml > w3c-ecma/test416.txml | `TestSCXML/w3c-ecma/test416.txml` | ported |  |
| scxml > w3c-ecma/test417.txml | `TestSCXML/w3c-ecma/test417.txml` | ported |  |
| scxml > w3c-ecma/test419.txml | `TestSCXML/w3c-ecma/test419.txml` | ported |  |
| scxml > w3c-ecma/test421.txml | `TestSCXML/w3c-ecma/test421.txml` | ported |  |
| scxml > w3c-ecma/test422.txml | `TestSCXML/w3c-ecma/test422.txml` | skipped-in-JS | commented out in JS: conversion of type-less <invoke> not implemented yet |
| scxml > w3c-ecma/test423.txml | `TestSCXML/w3c-ecma/test423.txml` | ported |  |
| scxml > w3c-ecma/test436.txml | `TestSCXML/w3c-ecma/test436.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet + null datamodel not implemented yet |
| scxml > w3c-ecma/test444.txml | `TestSCXML/w3c-ecma/test444.txml` | skipped-in-JS | commented out in JS: datamodel being mutated in cond's expression 😱 |
| scxml > w3c-ecma/test445.txml | `TestSCXML/w3c-ecma/test445.txml` | ported |  |
| scxml > w3c-ecma/test446.txml | `TestSCXML/w3c-ecma/test446.txml` | skipped-in-JS | commented out in JS: conversion of <data src="..."> not implemented yet |
| scxml > w3c-ecma/test448.txml | `TestSCXML/w3c-ecma/test448.txml` | skipped-in-JS | commented out in JS: nested datamodels not implemented yet |
| scxml > w3c-ecma/test449.txml | `TestSCXML/w3c-ecma/test449.txml` | ported |  |
| scxml > w3c-ecma/test451.txml | `TestSCXML/w3c-ecma/test451.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet |
| scxml > w3c-ecma/test452.txml | `TestSCXML/w3c-ecma/test452.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test453.txml | `TestSCXML/w3c-ecma/test453.txml` | ported |  |
| scxml > w3c-ecma/test456.txml | `TestSCXML/w3c-ecma/test456.txml` | skipped-in-JS | commented out in JS: conversion of <script> not implemented yet |
| scxml > w3c-ecma/test457.txml | `TestSCXML/w3c-ecma/test457.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test459.txml | `TestSCXML/w3c-ecma/test459.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test460.txml | `TestSCXML/w3c-ecma/test460.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test487.txml | `TestSCXML/w3c-ecma/test487.txml` | skipped-in-JS | commented out in JS: this has a syntax error on purpose, so it's not included |
| scxml > w3c-ecma/test488.txml | `TestSCXML/w3c-ecma/test488.txml` | skipped-in-JS | commented out in JS: error.execution when evaluating param |
| scxml > w3c-ecma/test495.txml | `TestSCXML/w3c-ecma/test495.txml` | ported |  |
| scxml > w3c-ecma/test496.txml | `TestSCXML/w3c-ecma/test496.txml` | skipped-in-JS | commented out in JS: error.communication not implemented yet |
| scxml > w3c-ecma/test500.txml | `TestSCXML/w3c-ecma/test500.txml` | skipped-in-JS | commented out in JS: _ioprocessors not yet available for expressions |
| scxml > w3c-ecma/test501.txml | `TestSCXML/w3c-ecma/test501.txml` | skipped-in-JS | commented out in JS: _ioprocessors not yet available for expressions |
| scxml > w3c-ecma/test503.txml | `TestSCXML/w3c-ecma/test503.txml` | ported |  |
| scxml > w3c-ecma/test504.txml | `TestSCXML/w3c-ecma/test504.txml` | ported |  |
| scxml > w3c-ecma/test505.txml | `TestSCXML/w3c-ecma/test505.txml` | ported |  |
| scxml > w3c-ecma/test506.txml | `TestSCXML/w3c-ecma/test506.txml` | skipped-in-JS | commented out in JS: `reenter` semantics in v5 are different from SCXML type="internal"/"external" transitions, we respect `reenter` on all state types, not just on compound states |
| scxml > w3c-ecma/test509.txml | `TestSCXML/w3c-ecma/test509.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test510.txml | `TestSCXML/w3c-ecma/test510.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test518.txml | `TestSCXML/w3c-ecma/test518.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test519.txml | `TestSCXML/w3c-ecma/test519.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test520.txml | `TestSCXML/w3c-ecma/test520.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test521.txml | `TestSCXML/w3c-ecma/test521.txml` | skipped-in-JS | commented out in JS: error.communication not implemented yet |
| scxml > w3c-ecma/test522.txml | `TestSCXML/w3c-ecma/test522.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test525.txml | `TestSCXML/w3c-ecma/test525.txml` | skipped-in-JS | commented out in JS: <foreach> not implemented yet |
| scxml > w3c-ecma/test527.txml | `TestSCXML/w3c-ecma/test527.txml` | skipped-in-JS | commented out in JS: conversion of <donedata> not implemented yet |
| scxml > w3c-ecma/test528.txml | `TestSCXML/w3c-ecma/test528.txml` | skipped-in-JS | commented out in JS: conversion of <donedata> not implemented yet + error.execution when evaluating donedata |
| scxml > w3c-ecma/test529.txml | `TestSCXML/w3c-ecma/test529.txml` | skipped-in-JS | commented out in JS: conversion of <donedata> not implemented yet |
| scxml > w3c-ecma/test530.txml | `TestSCXML/w3c-ecma/test530.txml` | skipped-in-JS | commented out in JS: https://github.com/davidkpiano/xstate/pull/1811#discussion_r551897693 |
| scxml > w3c-ecma/test531.txml | `TestSCXML/w3c-ecma/test531.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test532.txml | `TestSCXML/w3c-ecma/test532.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test533.txml | `TestSCXML/w3c-ecma/test533.txml` | skipped-in-JS | commented out in JS: we allow `reenter: false` to not leave the source state even if that source state is not compound |
| scxml > w3c-ecma/test534.txml | `TestSCXML/w3c-ecma/test534.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test550.txml | `TestSCXML/w3c-ecma/test550.txml` | skipped-in-JS | commented out in JS: non-root datamodel with early binding not implemented yet |
| scxml > w3c-ecma/test551.txml | `TestSCXML/w3c-ecma/test551.txml` | skipped-in-JS | commented out in JS: non-root datamodel with early binding not implemented yet |
| scxml > w3c-ecma/test552.txml | `TestSCXML/w3c-ecma/test552.txml` | skipped-in-JS | commented out in JS: conversion of <data src="..."> not implemented yet |
| scxml > w3c-ecma/test553.txml | `TestSCXML/w3c-ecma/test553.txml` | skipped-in-JS | commented out in JS: namelist not implemented yet + errored send not dispatching an event |
| scxml > w3c-ecma/test554.txml | `TestSCXML/w3c-ecma/test554.txml` | skipped-in-JS | commented out in JS: namelist not implemented yet + errored invoke cancelled |
| scxml > w3c-ecma/test557.txml | `TestSCXML/w3c-ecma/test557.txml` | skipped-in-JS | commented out in JS: conversion of <data src="..."> not implemented yet |
| scxml > w3c-ecma/test558.txml | `TestSCXML/w3c-ecma/test558.txml` | skipped-in-JS | commented out in JS: conversion of <data src="..."> not implemented yet |
| scxml > w3c-ecma/test560.txml | `TestSCXML/w3c-ecma/test560.txml` | ported |  |
| scxml > w3c-ecma/test561.txml | `TestSCXML/w3c-ecma/test561.txml` | skipped-in-JS | commented out in JS: processor creates an ECMAScript DOM object _event.data when receiving XML in an event |
| scxml > w3c-ecma/test562.txml | `TestSCXML/w3c-ecma/test562.txml` | skipped-in-JS | commented out in JS: test that processor creates space normalized string in _event.data when receiving anything other than KVPs or XML in an event |
| scxml > w3c-ecma/test567.txml | `TestSCXML/w3c-ecma/test567.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test569.txml | `TestSCXML/w3c-ecma/test569.txml` | skipped-in-JS | commented out in JS: _ioprocessors not yet available for expressions |
| scxml > w3c-ecma/test570.txml | `TestSCXML/w3c-ecma/test570.txml` | ported |  |
| scxml > w3c-ecma/test576.txml | `TestSCXML/w3c-ecma/test576.txml` | skipped-in-JS | commented out in JS: multiple initial states are not supported |
| scxml > w3c-ecma/test577.txml | `TestSCXML/w3c-ecma/test577.txml` | skipped-in-JS | commented out in JS: Basic HTTP Event I/O processor not implemented |
| scxml > w3c-ecma/test578.txml | `TestSCXML/w3c-ecma/test578.txml` | skipped-in-JS | commented out in JS: conversion of <content> in <send> not implemented yet |
| scxml > w3c-ecma/test579.txml | `TestSCXML/w3c-ecma/test579.txml` | skipped-in-JS | commented out in JS: executable content in history states not implemented yet |
| scxml > w3c-ecma/test580.txml | `TestSCXML/w3c-ecma/test580.txml` | skipped-in-JS | commented out in JS: conversion of In() predicate not implemented yet |

## Harness mapping

- `toMachine(xml)` → `scxml.ToMachine(xml string) *xs.StateMachine[map[string]any]` (context = SCXML datamodel); `sanitizeStateId` → `scxml.SanitizeStateID`. Stubs in `scxml/scxml.go` panic with `xstate/scxml: not implemented`.
- `runW3TestToCompletion` → `scxmlRunW3TestToCompletion`: actor with a no-op `xs.WithLogger`, real timers, resolves when the completed snapshot value is `"final"` or `"pass"`, otherwise fails with `Reached "fail" state from state <JSON>`. The JS promise is bounded by vitest's default 5 s timeout; Go waits at most 5 s (`scxmlW3TestTimeout`).
- `runTestToCompletion` → `scxmlRunTestToCompletion`: `xs.NewSimulatedClock()` + `xs.WithClock`, `clock.Increment(after ms)`, `xs.GetStateNodes(machine.Root, value)` ids must contain `SanitizeStateID(nextConfiguration[0])` (`assert.Contains`). The JS `throw` inside the `complete` observer on a `"fail"` state becomes `t.Errorf` and leaves `done` false, as in JS.
- Fixtures: `scxml/testdata/<group>/<name>.{scxml,json}` for the 169 enabled cases only (338 files), copied verbatim from `references/scion/package/test`; the override `assign-current-small-step/test0.scxml` comes from `references/xstate/packages/core/test/fixtures/scxml`. Skipped cases have no fixture copied.

## API gaps

None. The harness only uses contract APIs (`xs.CreateActor`, `xs.WithLogger`, `xs.WithClock`, `xs.NewSimulatedClock`, `xs.GetStateNodes`, `StateMachine.Root`, `StateNode.ID`), so `apigap_scxml.go` was not created. New package API: `scxml.ToMachine`, `scxml.SanitizeStateID` (`scxml/scxml.go`).

## Reviewer notes

- JS `toMachine(xml)` takes no options; `ToMachine` mirrors that signature and panics on an unconvertible document (JS throws), matching `xs.CreateMachine`. Add options only when the implementation needs them.
- Not ported: the `catch` block that `console.log`s `JSON.stringify(machine.config)` on failure. It is debug output, not an assertion.
- `onlyTests` (empty in JS) is not ported; `go test -run 'TestSCXML/<group>/<name>'` serves the same purpose.
- `send-idlocation/test0`, `send-internal/test0` and `data/data_invalid` are commented out in JS without a reason of their own; `scxml-prefix-event-name-matching/test0`, `test1` take their reason from the comment line above them (`prefix event matching not implemented yet`).
