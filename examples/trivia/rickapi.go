package trivia

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
)

// RickAndMortyAPI mirrors RICK_AND_MORTY_API in src/common/constants.ts.
const RickAndMortyAPI = "https://rickandmortyapi.com/api/character"

// Character mirrors RMCharacter in src/common/types.ts.
type Character struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Species string   `json:"species"`
	Gender  string   `json:"gender"`
	Image   string   `json:"image"`
	Episode []string `json:"episode"`
}

// Episode mirrors RMEpisode in src/common/types.ts.
type Episode struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	AirDate    string   `json:"air_date"`
	Episode    string   `json:"episode"`
	Characters []string `json:"characters"`
	URL        string   `json:"url"`
	Created    string   `json:"created"`
}

// RickAPI mirrors RickCharactersImpl in src/services/RickApi.tsx. Every method
// follows the JS `.catch((err) => console.log(err))`: a failed request is
// reported through Log and yields the zero result (JS resolves with undefined).
type RickAPI struct {
	BaseURL string
	Client  *http.Client
	// Random returns a float in [0, 1) like Math.random.
	Random func() float64
	Log    func(err error)
}

// NewRickAPI returns the client for the public API with Math.random-like randomness.
func NewRickAPI() *RickAPI {
	return &RickAPI{
		BaseURL: RickAndMortyAPI,
		Client:  http.DefaultClient,
		Random:  rand.Float64,
		Log:     func(err error) { log.Println(err) },
	}
}

// RandomNumber mirrors getRandomNumber in src/common/constants.ts: floor(random * 400).
func (r *RickAPI) RandomNumber() int { return int(r.Random() * 400) }

func (r *RickAPI) get(ctx context.Context, url string, out any) bool {
	err := r.fetch(ctx, url, out)
	if err != nil {
		r.Log(err)
		return false
	}
	return true
}

func (r *RickAPI) fetch(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request to %s failed with status code %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetCharacters mirrors getCharacters(page): GET <base>/?page=<page>, the `results` array.
func (r *RickAPI) GetCharacters(ctx context.Context, page int) []Character {
	var body struct {
		Results []Character `json:"results"`
	}
	if !r.get(ctx, fmt.Sprintf("%s/?page=%d", r.BaseURL, page), &body) {
		return nil
	}
	return body.Results
}

// GetCharacter mirrors getCharacter(id): GET <base>/<id>. Nil on failure.
func (r *RickAPI) GetCharacter(ctx context.Context, id int) *Character {
	var c Character
	if !r.get(ctx, fmt.Sprintf("%s/%d", r.BaseURL, id), &c) {
		return nil
	}
	return &c
}

// GetRandomCharacters mirrors getRandomCharacters(): GET <base>/<n>,<n>,<n> with three random ids.
func (r *RickAPI) GetRandomCharacters(ctx context.Context) []Character {
	ids := []string{
		fmt.Sprint(r.RandomNumber()), fmt.Sprint(r.RandomNumber()), fmt.Sprint(r.RandomNumber()),
	}
	var cs []Character
	if !r.get(ctx, r.BaseURL+"/"+strings.Join(ids, ","), &cs) {
		return nil
	}
	return cs
}

// GetClue mirrors getClue(episode): GET the episode URL. Nil on failure.
func (r *RickAPI) GetClue(ctx context.Context, episodeURL string) *Episode {
	var e Episode
	if !r.get(ctx, episodeURL, &e) {
		return nil
	}
	return &e
}
