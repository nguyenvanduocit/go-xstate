// Package donut ports references/xstate/examples/persisted-donut-maker/donutMachine.ts
// and the console entry main.ts (restore the persisted snapshot, print the state and
// the next events after every snapshot, persist the snapshot to a JSON file).
package donut

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the donut machine's extended state: the machine declares none, so
// the JS snapshot carries `context: {}`.
type Context struct{}

// Machine mirrors donutMachine.
func Machine() *xs.StateMachine[Context] {
	mixing := func(event, target string) xs.StateConfig {
		return xs.StateConfig{
			Key: "mixing",
			On:  map[string]xs.Transitions{event: {{Target: target}}},
		}
	}
	region := func(key, event string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "mixing",
			States: xs.States{
				mixing(event, "mixed"),
				{Key: "mixed", Type: xs.Final},
			},
		}
	}
	next := func(key, target string) xs.StateConfig {
		return xs.StateConfig{Key: key, On: map[string]xs.Transitions{"NEXT": {{Target: target}}}}
	}
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "donut",
		Initial: "ingredients",
		States: xs.States{
			next("ingredients", "directions"),
			{
				Key:     "directions",
				Initial: "makeDough",
				OnDone:  xs.Transitions{{Target: "fry"}},
				States: xs.States{
					next("makeDough", "mix"),
					{
						Key:    "mix",
						Type:   xs.Parallel,
						OnDone: xs.Transitions{{Target: "allMixed"}},
						States: xs.States{
							region("mixDry", "MIXED_DRY"),
							region("mixWet", "MIXED_WET"),
						},
					},
					{Key: "allMixed", Type: xs.Final},
				},
			},
			next("fry", "flip"),
			next("flip", "dry"),
			next("dry", "glaze"),
			next("glaze", "serve"),
			{Key: "serve", On: map[string]xs.Transitions{"ANOTHER_DONUT": {{Target: "ingredients"}}}},
		},
	})
}

// NextEvents mirrors __unsafe_getAllOwnEventDescriptors(snapshot): the own event
// descriptors of the active state nodes, de-duplicated, in node order.
func NextEvents(s *xs.MachineSnapshot[Context]) []string {
	seen := map[string]bool{}
	events := []string{}
	for _, n := range s.StateNodes() {
		for _, e := range n.OwnEvents() {
			if !seen[e] {
				seen[e] = true
				events = append(events, e)
			}
		}
	}
	return events
}

// bold mirrors the `\x1b[1m...\x1b[0m` wrapping in main.ts.
func bold(s string) string { return "\x1b[1m" + s + "\x1b[0m" }

// FormatSnapshot returns what main.ts passes to console.log for one snapshot,
// joined the way console.log joins its arguments (a space), plus the newline.
func FormatSnapshot(s *xs.MachineSnapshot[Context]) (string, error) {
	value, err := json.Marshal(s.Value)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range NextEvents(s) {
		if !strings.HasPrefix(e, "done.") {
			b.WriteString("\n  " + bold(e))
		}
	}
	return strings.Join([]string{
		"Current state:",
		bold(string(value)) + "\n",
		"Next events:",
		b.String(),
		"\nEnter the next event to send:",
	}, " ") + "\n", nil
}

// Run mirrors main.ts. stateFile stands for './persisted-state.json'. The persisted
// snapshot is restored from it when it exists and parses; every snapshot is
// written back. Each line of stdin (trimmed) is sent as an event type. Run returns
// when stdin is exhausted.
func Run(w io.Writer, stdin io.Reader, stateFile string) error {
	var restored any
	if raw, err := os.ReadFile(stateFile); err != nil || json.Unmarshal(raw, &restored) != nil {
		fmt.Fprintln(w, "No persisted state found.")
		restored = nil
	}

	var opts []xs.ActorOption
	if restored != nil {
		opts = append(opts, xs.WithSnapshot(restored))
	}
	actor := xs.CreateActor(Machine(), opts...)

	var runErr error
	fail := func(err error) {
		if runErr == nil {
			runErr = err
		}
	}
	sub := actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) {
			text, err := FormatSnapshot(s)
			if err != nil {
				fail(err)
				return
			}
			if _, err := io.WriteString(w, text); err != nil {
				fail(err)
			}
			persisted, err := json.Marshal(actor.GetPersistedSnapshot())
			if err != nil {
				fail(err)
				return
			}
			if err := os.WriteFile(stateFile, persisted, 0o644); err != nil {
				fail(err)
			}
		},
		Complete: func() {
			fmt.Fprintln(w, "workflow completed", actor.GetSnapshot().Output)
		},
	})
	actor.Start()
	// The JS process just exits when stdin ends, so the observer never sees the
	// completion that Stop() would deliver; unsubscribe before stopping.
	defer actor.Stop()
	defer sub.Unsubscribe()

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		actor.Send(xs.Ev(strings.TrimSpace(scanner.Text())))
	}
	if err := scanner.Err(); err != nil {
		fail(err)
	}
	return runErr
}
