// Package trivia ports references/xstate/examples/trivia-game-example/src/triviaMachine.ts
// together with the non-UI helpers it uses (src/services/RickApi.tsx,
// src/common/constants.ts, src/common/types.ts).
package trivia

import (
	"context"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the trivia machine's extended state.
type Context struct {
	HomePageCharacters []Character `json:"homePageCharacters"`
	HasLoaded          bool        `json:"hasLoaded"`
	CurrentCharacter   *Character  `json:"currentCharacter"`
	RandomCharacters   []Character `json:"randomCharacters"`
	IsClueOpened       bool        `json:"isClueOpened"`
	Points             int         `json:"points"`
	Question           int         `json:"question"`
	Lifes              int         `json:"lifes"`
}

func update(fn func(c *Context, a xs.AssignArgs[Context])) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		c := a.Context
		fn(&c, a)
		return c
	})
}

// selectedAnswer reads `event.answer` (a JSON number in golden traces, so float64).
func selectedAnswer(e xs.Event) (int, bool) {
	ev, ok := e.(xs.E)
	if !ok || ev.EventType() != "user.selectAnswer" {
		return 0, false
	}
	switch n := ev["answer"].(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
}

var (
	isAnswerCorrect = xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
		answer, ok := selectedAnswer(a.Event)
		if !ok {
			panic("Expected event to have type user.selectAnswer")
		}
		if a.Context.CurrentCharacter == nil {
			return false
		}
		return answer == a.Context.CurrentCharacter.ID
	})
	hasLostGame = xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Lifes <= 0 })
	hasWonGame  = xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Points >= 100 })
)

func checkGameOver() xs.Transitions {
	return xs.Transitions{
		{Guard: xs.GuardRef{Type: "hasLostGame"}, Target: "lostGame"},
		{Guard: xs.GuardRef{Type: "hasWonGame"}, Target: "wonGame"},
	}
}

// ActorNames lists the three promise actors the machine invokes by name; a trace
// can swap them with Machine().Provide, like machine.provide({actors}) in JS.
const (
	LoadHomePageCharacters = "loadHomePageCharacters"
	LoadSingleCharacter    = "loadSingleCharacter"
	LoadRandomCharacters   = "loadRandomCharacters"
)

// NewMachine mirrors triviaMachine with the promise actors backed by api.
func NewMachine(api *RickAPI) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Guards: map[string]xs.Guard{
			"isAnswerCorrect": isAnswerCorrect,
			"hasLostGame":     hasLostGame,
			"hasWonGame":      hasWonGame,
		},
		Actions: map[string]xs.Action{
			"goToTriviaPage": xs.ActionFunc(func(xs.ActionArgs[Context]) {}),
			"resetTriviaData": update(func(c *Context, _ xs.AssignArgs[Context]) {
				c.CurrentCharacter = nil
				c.RandomCharacters = []Character{}
				c.Points = 0
				c.Question = 0
				c.Lifes = 3
			}),
		},
		Actors: map[string]xs.ActorLogic{
			LoadHomePageCharacters: xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) ([]Character, error) {
				return api.GetCharacters(ctx, int(api.Random()*34)), nil
			}),
			LoadSingleCharacter: xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (*Character, error) {
				return api.GetCharacter(ctx, api.RandomNumber()), nil
			}),
			LoadRandomCharacters: xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) ([]Character, error) {
				return api.GetRandomCharacters(ctx), nil
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "triviaMachine",
		Initial: "homepage",
		Context: Context{
			HomePageCharacters: []Character{},
			HasLoaded:          false,
			CurrentCharacter:   nil,
			RandomCharacters:   []Character{},
			IsClueOpened:       false,
			Points:             0,
			Question:           0,
			Lifes:              3,
		},
		States: xs.States{
			{
				Key:     "homepage",
				Initial: "loadingData",
				States: xs.States{
					{
						Key: "loadingData",
						Invoke: []xs.InvokeConfig{{
							Src: LoadHomePageCharacters,
							OnDone: xs.Transitions{{
								Target: "dataLoaded",
								Actions: xs.Actions{update(func(c *Context, a xs.AssignArgs[Context]) {
									c.HomePageCharacters, _ = a.Event.(xs.DoneActorEvent).Output.([]Character)
									c.HasLoaded = true
								})},
							}},
						}},
					},
					{
						Key: "dataLoaded",
						On:  map[string]xs.Transitions{"user.play": {{Target: "#instructionModal"}}},
					},
				},
			},
			{
				Key: "instructionModal",
				ID:  "instructionModal",
				On: map[string]xs.Transitions{
					"user.close":  {{Target: "homepage.dataLoaded"}},
					"user.reject": {{Target: "homepage.dataLoaded"}},
					"user.accept": {{
						Target: "startTrivia",
						Actions: xs.Actions{update(func(c *Context, _ xs.AssignArgs[Context]) {
							c.HasLoaded = false
						})},
					}},
				},
			},
			{
				Key:     "startTrivia",
				ID:      "startTrivia",
				Initial: "loadQuestionData",
				Entry:   xs.Actions{xs.ActionRef{Type: "goToTriviaPage"}, xs.ActionRef{Type: "resetTriviaData"}},
				States: xs.States{
					{
						Key:     "loadQuestionData",
						ID:      "loadQuestionData",
						Initial: "loadCharacter",
						Entry: xs.Actions{update(func(c *Context, _ xs.AssignArgs[Context]) {
							c.HasLoaded = false
						})},
						States: xs.States{
							{
								Key: "loadCharacter",
								Invoke: []xs.InvokeConfig{{
									Src: LoadSingleCharacter,
									OnDone: xs.Transitions{{
										Target: "loadRandomCharacters",
										Actions: xs.Actions{update(func(c *Context, a xs.AssignArgs[Context]) {
											c.CurrentCharacter, _ = a.Event.(xs.DoneActorEvent).Output.(*Character)
										})},
									}},
								}},
							},
							{
								Key: "loadRandomCharacters",
								Invoke: []xs.InvokeConfig{{
									Src: LoadRandomCharacters,
									OnDone: xs.Transitions{{
										Target: "#questionReady",
										Actions: xs.Actions{update(func(c *Context, a xs.AssignArgs[Context]) {
											c.RandomCharacters, _ = a.Event.(xs.DoneActorEvent).Output.([]Character)
											c.Question++
											c.HasLoaded = true
										})},
									}},
								}},
							},
						},
					},
					{
						Key:     "questionReady",
						ID:      "questionReady",
						Initial: "questionStart",
						On: map[string]xs.Transitions{
							"user.toggleClue": {{Actions: xs.Actions{update(func(c *Context, _ xs.AssignArgs[Context]) {
								c.IsClueOpened = !c.IsClueOpened
							})}}},
						},
						States: xs.States{
							{
								Key: "questionStart",
								On: map[string]xs.Transitions{
									"user.selectAnswer": {
										{Target: "correctAnswer", Guard: xs.GuardRef{Type: "isAnswerCorrect"}},
										{Target: "incorrectAnswer", Guard: xs.Not(xs.GuardRef{Type: "isAnswerCorrect"})},
									},
								},
							},
							{
								Key: "correctAnswer",
								Entry: xs.Actions{update(func(c *Context, _ xs.AssignArgs[Context]) {
									c.Points += 10
								})},
								Always: checkGameOver(),
								On:     map[string]xs.Transitions{"user.nextQuestion": {{Target: "#loadQuestionData"}}},
							},
							{
								Key: "incorrectAnswer",
								Entry: xs.Actions{update(func(c *Context, _ xs.AssignArgs[Context]) {
									c.Lifes--
								})},
								Always: checkGameOver(),
								On:     map[string]xs.Transitions{"user.nextQuestion": {{Target: "#loadQuestionData"}}},
							},
							{Key: "lostGame", On: map[string]xs.Transitions{"user.playAgain": {{Target: "#startTrivia"}}}},
							{Key: "wonGame", On: map[string]xs.Transitions{"user.playAgain": {{Target: "#startTrivia"}}}},
						},
					},
				},
			},
		},
	})
}

// Machine mirrors triviaMachine with the real Rick and Morty API.
func Machine() *xs.StateMachine[Context] { return NewMachine(NewRickAPI()) }
