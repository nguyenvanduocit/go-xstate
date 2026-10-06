package persistedstate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

const bold = "\x1b[1m%s\x1b[0m"

// jsonText renders v like JSON.stringify (keys of Go maps are sorted, no HTML escaping).
func jsonText(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// toDocument converts a persisted snapshot into the JSON-shaped document stored in MongoDB.
func toDocument(persisted any) (map[string]any, error) {
	raw, err := json.Marshal(persisted)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	err = json.Unmarshal(raw, &doc)
	return doc, err
}

// nextEvents mirrors __unsafe_getAllOwnEventDescriptors(snapshot) minus the `done.` events.
func nextEvents(snapshot Snapshot) []string {
	seen := map[string]bool{}
	var out []string
	for _, node := range snapshot.StateNodes() {
		for _, event := range node.OwnEvents() {
			if seen[event] || strings.HasPrefix(event, "done.") {
				continue
			}
			seen[event] = true
			out = append(out, event)
		}
	}
	return out
}

// Run mirrors main.ts. It connects to store, restores the donut actor from the
// persisted snapshot (if any), saves the persisted snapshot after every snapshot
// the actor emits, prints the current state and the events that can be sent next,
// and sends one event per line read from in. It returns when in reaches EOF
// (main.ts keeps listening until the process is killed) or when the actor completes.
//
// Errors from Connect, FindOne or restoring are printed as `error details:` and end the
// run, as the catch block of main.ts does. A failed save is printed the same way and
// the run continues.
func Run(ctx context.Context, w io.Writer, in io.Reader, store Store) {
	printf := func(format string, args ...any) { fmt.Fprintf(w, format+"\n", args...) }
	fail := func(err error) { printf("error details:  Error: %s", err) }

	if err := store.Connect(ctx); err != nil {
		fail(err)
		return
	}
	restored, err := store.FindOne(ctx)
	if err != nil {
		fail(err)
		return
	}
	if restored == nil {
		printf("no persisted state found in db. starting from scratch.")
	}
	printf("restored state:  %s", jsonText(restored))

	var opts []xs.ActorOption
	if persisted, ok := restored["persistedState"]; ok {
		opts = append(opts, xs.WithSnapshot(persisted))
	}
	actor := xs.CreateActor(DonutMachine(), opts...)
	queue := &TaskQueue{}

	completed := false
	subscription := actor.Subscribe(xs.Observer[Snapshot]{
		Next: func(snapshot Snapshot) {
			queue.AddTask(func() {
				doc, err := toDocument(actor.GetPersistedSnapshot())
				if err == nil {
					var result UpdateResult
					result, err = store.UpdateOne(ctx, doc)
					if err == nil && (result.ModifiedCount > 0 || result.UpsertedCount > 0) {
						printf("persisted state saved to db.  %s", jsonText(result))
					}
				}
				if err != nil {
					fail(err)
					return
				}
				events := nextEvents(snapshot)
				var list strings.Builder
				for _, event := range events {
					list.WriteString("\n  " + fmt.Sprintf(bold, event))
				}
				printf("Current state: %s\n Next events: %s \nEnter the next event to send:",
					fmt.Sprintf(bold, jsonText(snapshot.Value)), list.String())
			})
		},
		Complete: func() {
			queue.AddTask(func() {
				completed = true
				printf("workflow completed %s", jsonText(actor.GetSnapshot().Output))
				if err := store.Close(ctx); err != nil {
					fail(err)
				}
			})
		},
	})
	defer actor.Stop()
	defer subscription.Unsubscribe()

	actor.Start()

	lines := bufio.NewScanner(in)
	for !completed && lines.Scan() {
		actor.Send(xs.Ev(strings.TrimSpace(lines.Text())))
	}
}
