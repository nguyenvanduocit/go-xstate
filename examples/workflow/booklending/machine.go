// Package booklending ports references/xstate/examples/workflow-book-lending/main.ts
// (serverless workflow "book lending").
package booklending

import (
	"context"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Lender mirrors the JS `Lender` interface.
type Lender struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

// Book mirrors context.book: the requested book plus its lending status
// ('onloan' | 'available' | 'unknown', or whatever the status actor answers).
type Book struct {
	Title  string `json:"title"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Context is the machine's extended state. Both fields start nil (JS null).
// Lender is never assigned: the JS machine ignores event.lender.
type Context struct {
	Book   *Book   `json:"book"`
	Lender *Lender `json:"lender"`
}

// Inputs of the invoked actors, in the key order of the JS input objects.
type (
	// BookInput is the input of 'Get status for book' and 'Check out book with id'.
	BookInput struct {
		BookID string `json:"bookid"`
	}
	// MessageInput is the input of 'Send status to lender'.
	MessageInput struct {
		BookID  string `json:"bookid"`
		Message string `json:"message"`
	}
	// LenderInput is the input of 'Request hold for lender', 'Cancel hold request for lender'
	// and 'Notify Lender for checkout'.
	LenderInput struct {
		BookID string  `json:"bookid"`
		Lender *Lender `json:"lender"`
	}
)

// StatusOutput is the output of 'Get status for book': `{ status }`.
type StatusOutput struct {
	Status string `json:"status"`
}

// LogFunc stands in for the actors' console.log(label, input).
type LogFunc func(label string, input any)

// BookLendingRequest builds the `bookLendingRequest` event.
func BookLendingRequest(title, id string, lender Lender) xs.E {
	return xs.E{
		"type":   "bookLendingRequest",
		"book":   map[string]any{"title": title, "id": id},
		"lender": lender,
	}
}

// requestedBook mirrors `{ ...event.book, status: 'unknown' }`. The event may come from Go
// code (BookLendingRequest) or from decoded JSON; both carry title and id.
func requestedBook(event xs.Event) *Book {
	ref := event.(xs.E)["book"].(map[string]any)
	title, _ := ref["title"].(string)
	id, _ := ref["id"].(string)
	return &Book{Title: title, ID: id, Status: "unknown"}
}

// delay mirrors the JS delay(ms) helper called with errorProbability 0. Cancelling ctx
// stands in for abandoning the promise when the invoking state exits.
func delay(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

// loggedCall mirrors the six fromPromise actors: console.log('Starting <name>', input),
// wait d (1000 ms in JS), resolve with out.
func loggedCall[O any](log LogFunc, label string, d time.Duration, out O) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (O, error) {
		log(label, a.Input)
		return out, delay(ctx, d)
	})
}

// Actors mirrors the actors implementation of the JS machine: each logs through log and
// waits d (1000 ms in JS). 'Get status for book' always answers 'available'.
func Actors(log LogFunc, d time.Duration) map[string]xs.ActorLogic {
	return map[string]xs.ActorLogic{
		"Get status for book":            loggedCall(log, "Starting Get status for book", d, StatusOutput{Status: "available"}),
		"Send status to lender":          loggedCall[any](log, "Starting Send status to lender", d, nil),
		"Request hold for lender":        loggedCall[any](log, "Starting Request hold for lender", d, nil),
		"Cancel hold request for lender": loggedCall[any](log, "Starting Cancel hold request for lender", d, nil),
		"Check out book with id":         loggedCall[any](log, "Starting Check out book with id", d, nil),
		"Notify Lender for checkout":     loggedCall[any](log, "Starting Notify Lender for checkout", d, nil),
	}
}

func bookInput(a xs.ExprArgs[Context]) any { return BookInput{BookID: a.Context.Book.ID} }

func lenderInput(a xs.ExprArgs[Context]) any {
	return LenderInput{BookID: a.Context.Book.ID, Lender: a.Context.Lender}
}

func bookStatusIs(status string) xs.Guard {
	return xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Book.Status == status })
}

// NewMachine mirrors `workflow` with its actors built by Actors(log, d).
// Delay 'PT2W' has no implementation, exactly as in JS: the after event is queued at
// once (see NOTES.md); Provide a Delays entry to make it a timer.
func NewMachine(log LogFunc, d time.Duration) *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		Initial: "Book Lending Request",
		Context: Context{},
		States: xs.States{
			{
				Key: "Book Lending Request",
				On: map[string]xs.Transitions{
					"bookLendingRequest": {{
						Target: "Get Book Status",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.Book = requestedBook(a.Event)
							return c
						})},
					}},
				},
			},
			{
				Key: "Get Book Status",
				Invoke: []xs.InvokeConfig{{
					Src:   "Get status for book",
					Input: xs.NewExpr(bookInput),
					OnDone: xs.Transitions{{
						Target: "Book Status Decision",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							book := *c.Book
							book.Status = a.Event.(xs.DoneActorEvent).Output.(StatusOutput).Status
							c.Book = &book
							return c
						})},
					}},
				}},
			},
			{
				Key: "Book Status Decision",
				Always: xs.Transitions{
					{Guard: bookStatusIs("onloan"), Target: "Report Status To Lender"},
					{Guard: bookStatusIs("available"), Target: "Check Out Book"},
					{Target: "End"},
				},
			},
			{
				Key: "Report Status To Lender",
				Invoke: []xs.InvokeConfig{{
					Src: "Send status to lender",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return MessageInput{
							BookID:  a.Context.Book.ID,
							Message: "Book " + a.Context.Book.Title + " is already on loan",
						}
					}),
					OnDone: xs.Transitions{{Target: "Wait for Lender response"}},
				}},
			},
			{
				Key: "Wait for Lender response",
				On: map[string]xs.Transitions{
					"holdBook":        {{Target: "Request Hold"}},
					"declineBookhold": {{Target: "Cancel Request"}},
				},
			},
			{
				Key: "Request Hold",
				Invoke: []xs.InvokeConfig{{
					Src:    "Request hold for lender",
					Input:  xs.NewExpr(lenderInput),
					OnDone: xs.Transitions{{Target: "Sleep two weeks"}},
				}},
			},
			{
				Key: "Cancel Request",
				Invoke: []xs.InvokeConfig{{
					Src:    "Cancel hold request for lender",
					Input:  xs.NewExpr(lenderInput),
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{
				Key:   "Sleep two weeks",
				After: map[string]xs.Transitions{"PT2W": {{Target: "Get Book Status"}}},
			},
			{
				Key:     "Check Out Book",
				Initial: "Checking out book",
				States: xs.States{
					{
						Key: "Checking out book",
						Invoke: []xs.InvokeConfig{{
							Src:    "Check out book with id",
							Input:  xs.NewExpr(bookInput),
							OnDone: xs.Transitions{{Target: "Notifying Lender"}},
						}},
					},
					{
						Key: "Notifying Lender",
						Invoke: []xs.InvokeConfig{{
							Src:    "Notify Lender for checkout",
							Input:  xs.NewExpr(lenderInput),
							OnDone: xs.Transitions{{Target: "End"}},
						}},
					},
					{Key: "End", Type: xs.Final},
				},
			},
			{Key: "End", Type: xs.Final},
		},
	}, xs.Implementations{Actors: Actors(log, d)})
}
