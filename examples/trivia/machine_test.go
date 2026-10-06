package trivia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	trivia "github.com/nguyenvanduocit/go-xstate/examples/trivia"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func character(id int) trivia.Character {
	return trivia.Character{
		ID:      id,
		Name:    "Character " + strconv.Itoa(id),
		Status:  "Alive",
		Species: "Human",
		Gender:  "Male",
		Image:   "https://example.test/avatar/" + strconv.Itoa(id) + ".jpeg",
		Episode: []string{"https://example.test/episode/" + strconv.Itoa(id)},
	}
}

// TestTrace replays testdata/trivia-game-example.golden.json, recorded from the JS example by
// scripts/trace/trivia-game-example/trivia-game-example.ts. The three promise actors are
// replaced by the same deterministic stubs as in the JS script.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/trivia-game-example/src/triviaMachine.ts#L6
// JS trace: scripts/trace/trivia-game-example/trivia-game-example.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/trivia-game-example.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[trivia.Context]] {
		var singles atomic.Int32
		machine := trivia.Machine().Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{
			trivia.LoadHomePageCharacters: xs.FromPromise(func(context.Context, xs.PromiseArgs) ([]trivia.Character, error) {
				time.Sleep(10 * time.Millisecond) // same delay as the JS stub, see trivia-game-example.ts
				return []trivia.Character{character(1), character(2)}, nil
			}),
			trivia.LoadSingleCharacter: xs.FromPromise(func(context.Context, xs.PromiseArgs) (*trivia.Character, error) {
				time.Sleep(10 * time.Millisecond)
				c := character(100 + int(singles.Add(1)))
				return &c, nil
			}),
			trivia.LoadRandomCharacters: xs.FromPromise(func(context.Context, xs.PromiseArgs) ([]trivia.Character, error) {
				return []trivia.Character{character(201), character(202), character(203)}, nil
			}),
		}})
		return xs.CreateActor(machine, xs.WithClock(clock))
	})
}

// fakeAPI serves the three endpoints RickAPI calls and records the request paths.
func fakeAPI(t *testing.T, fail bool) (*trivia.RickAPI, *[]string) {
	t.Helper()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		if fail {
			http.Error(w, "nope", http.StatusInternalServerError)
			return
		}
		if strings.Contains(r.URL.Path, ",") { // /<a>,<b>,<c>
			w.Write([]byte(`[{"id":1},{"id":2},{"id":3}]`))
			return
		}
		switch r.URL.Path {
		case "/api/character/", "/api/character":
			w.Write([]byte(`{"info":{},"results":[{"id":1,"name":"Rick","episode":["e1"]},{"id":2,"name":"Morty","episode":[]}]}`))
		case "/api/character/42":
			w.Write([]byte(`{"id":42,"name":"Answer","episode":["e"]}`))
		case "/api/episode/7":
			w.Write([]byte(`{"id":7,"name":"Pilot","air_date":"December 2, 2013","episode":"S01E01"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	var logged []error
	api := &trivia.RickAPI{
		BaseURL: srv.URL + "/api/character",
		Client:  srv.Client(),
		Random:  func() float64 { return 0.0075 }, // floor(0.0075 * 400) = 3
		Log:     func(err error) { logged = append(logged, err) },
	}
	t.Cleanup(func() {
		if fail {
			assert.NotEmpty(t, logged)
		} else {
			assert.Empty(t, logged)
		}
	})
	return api, &paths
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/trivia-game-example/src/services/RickApi.tsx#L5
// Related JS trace: scripts/trace/trivia-game-example/trivia-game-example.ts.
func TestRickAPI(t *testing.T) {
	api, paths := fakeAPI(t, false)
	ctx := context.Background()

	cs := api.GetCharacters(ctx, 5)
	require.Len(t, cs, 2)
	assert.Equal(t, "Morty", cs[1].Name)

	c := api.GetCharacter(ctx, 42)
	require.NotNil(t, c)
	assert.Equal(t, 42, c.ID)

	assert.Len(t, api.GetRandomCharacters(ctx), 3)

	api.Random = func() float64 { return 0.999 }
	assert.Equal(t, 399, api.RandomNumber())

	e := api.GetClue(ctx, api.BaseURL[:len(api.BaseURL)-len("/character")]+"/episode/7")
	require.NotNil(t, e)
	assert.Equal(t, trivia.Episode{ID: 7, Name: "Pilot", AirDate: "December 2, 2013", Episode: "S01E01"}, *e)

	assert.Equal(t, []string{"/api/character/?page=5", "/api/character/42", "/api/character/3,3,3", "/api/episode/7"}, *paths)
}

// A failed request is logged and yields the zero value, like the JS `.catch(console.log)`.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/trivia-game-example/src/services/RickApi.tsx#L5
// Related JS trace: scripts/trace/trivia-game-example/trivia-game-example.ts.
func TestRickAPIFailure(t *testing.T) {
	api, _ := fakeAPI(t, true)
	ctx := context.Background()
	assert.Nil(t, api.GetCharacters(ctx, 1))
	assert.Nil(t, api.GetCharacter(ctx, 1))
	assert.Nil(t, api.GetRandomCharacters(ctx))
	assert.Nil(t, api.GetClue(ctx, api.BaseURL+"/1"))
}

// The real actors run end to end against the fake server: homepage -> instruction -> question.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/trivia-game-example/src/services/RickApi.tsx#L5
// Related JS trace: scripts/trace/trivia-game-example/trivia-game-example.ts.
func TestMachineWithRickAPI(t *testing.T) {
	api, _ := fakeAPI(t, false)
	api.Random = func() float64 { return 42.0 / 400 } // character 42
	actor := xs.CreateActor(trivia.NewMachine(api))
	actor.Start()
	defer actor.Stop()

	wait := func(path string) *xs.MachineSnapshot[trivia.Context] {
		s, err := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[trivia.Context]) bool {
			return s.Matches(path)
		}, xs.WaitForOptions{}).Wait()
		require.NoError(t, err)
		return s
	}
	s := wait("homepage.dataLoaded")
	assert.Len(t, s.Context.HomePageCharacters, 2)
	actor.Send(xs.Ev("user.play"))
	actor.Send(xs.Ev("user.accept"))
	s = wait("startTrivia.questionReady.questionStart")
	require.NotNil(t, s.Context.CurrentCharacter)
	assert.Equal(t, 42, s.Context.CurrentCharacter.ID)
	assert.Len(t, s.Context.RandomCharacters, 3)
	assert.Equal(t, 1, s.Context.Question)
}
