package booklending

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Run mirrors the entry of main.ts: one actor, a subscriber that prints every context and
// "workflow completed ..." on completion, then the bookLendingRequest event. The real 1 s
// actors print to w.
func Run(w io.Writer) error { return RunWith(context.Background(), w, time.Second) }

// RunWith is Run with the actors' delay injected. Like the Node process, it returns when
// nothing is pending any more: no invoked actor is running and the machine is not waiting in
// 'Sleep two weeks'. ctx cancellation also returns.
func RunWith(ctx context.Context, w io.Writer, d time.Duration) error {
	out := newConsole(w)
	actor := xs.CreateActor(NewMachine(out.log, d))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) { out.snapshot(s) },
		Complete: func() {
			// Stopping an unfinished actor on return also completes observers; only a done workflow prints.
			if snap := actor.GetSnapshot(); snap.Status == xs.StatusDone {
				out.println("workflow completed undefined")
			}
		},
	})
	actor.Start()
	defer actor.Stop()

	actor.Send(BookLendingRequest("The Hitchhiker's Guide to the Galaxy", "42",
		Lender{Name: "John Doe", Address: " ... ", Phone: " ... "}))

	_, err := xs.WaitFor(ctx, actor, func(s *xs.MachineSnapshot[Context]) bool {
		return s.Status != xs.StatusActive || (len(s.Children) == 0 && !s.Matches("Sleep two weeks"))
	}).Wait()
	return err
}

// console prints like console.log in the JS entry. In JS a promise actor's body runs
// synchronously when its state is entered, so its "Starting ..." line precedes the subscriber's
// print of that snapshot. Go promise bodies run on their own goroutine, so snapshot() waits
// until every actor newly invoked in the snapshot has logged before printing.
type console struct {
	mu       sync.Mutex
	cond     *sync.Cond
	w        io.Writer
	started  int
	expected int
	children map[string]bool
}

func newConsole(w io.Writer) *console {
	c := &console{w: w}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *console) println(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintln(c.w, s)
}

// log is the LogFunc of the real actors.
func (c *console) log(label string, input any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.w, "%s %s\n", label, inspect(input, 0))
	c.started++
	c.cond.Broadcast()
}

func (c *console) snapshot(s *xs.MachineSnapshot[Context]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := make(map[string]bool, len(s.Children))
	for id := range s.Children {
		next[id] = true
		if !c.children[id] {
			c.expected++
		}
	}
	c.children = next
	for c.started < c.expected {
		c.cond.Wait()
	}
	fmt.Fprintln(c.w, inspect(s.Context, 0))
}

// inspect renders a value the way Bun's console.log does: objects over several lines, two-space
// indent, trailing commas, double-quoted strings, null for nil. Structs print their json-tagged
// fields in declaration order; maps print their keys in sorted order.
func inspect(v any, depth int) string {
	return inspectValue(reflect.ValueOf(v), depth)
}

func inspectValue(v reflect.Value, depth int) string {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return "null"
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Invalid:
		return "undefined"
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Struct:
		t := v.Type()
		var lines []string
		for i := range t.NumField() {
			name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
			lines = append(lines, name+": "+inspectValue(v.Field(i), depth+1))
		}
		return block(lines, depth)
	case reflect.Map:
		var lines []string
		for _, k := range v.MapKeys() {
			lines = append(lines, k.String()+": "+inspectValue(v.MapIndex(k), depth+1))
		}
		sort.Strings(lines)
		return block(lines, depth)
	default:
		return fmt.Sprint(v.Interface())
	}
}

func block(lines []string, depth int) string {
	if len(lines) == 0 {
		return "{}"
	}
	pad := strings.Repeat("  ", depth+1)
	var b strings.Builder
	b.WriteString("{\n")
	for _, l := range lines {
		b.WriteString(pad + l + ",\n")
	}
	b.WriteString(strings.Repeat("  ", depth) + "}")
	return b.String()
}
