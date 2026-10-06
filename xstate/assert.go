package xstate

import (
	"fmt"
	"strings"
)

// AssertEvent mirrors assertEvent(event, type | types); panics on mismatch
// with the JS error message.
func AssertEvent(event Event, types ...string) {
	for _, t := range types {
		if matchesEventDescriptor(event.EventType(), t) {
			return
		}
	}
	var text string
	if len(types) == 1 {
		text = fmt.Sprintf(`type matching "%s"`, types[0])
	} else {
		text = fmt.Sprintf(`one of types matching "%s"`, strings.Join(types, `", "`))
	}
	s, ok := jsonStringify(event)
	if !ok {
		s = "[object Object]"
	}
	panic(fmt.Errorf("Expected event %s to have %s", s, text))
}
