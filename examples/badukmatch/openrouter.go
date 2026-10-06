package badukmatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenRouter calls the Decisions API, which both Clef and Jev support.
type OpenRouter struct {
	key      string
	endpoint string
	client   *http.Client
}

func NewOpenRouter(key string, timeout time.Duration) (*OpenRouter, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("OPENROUTER_KEY is required")
	}
	if timeout <= 0 {
		return nil, errors.New("request timeout must be positive")
	}
	return &OpenRouter{key: key, endpoint: "https://openrouter.ai/api/alpha/decisions", client: &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

type choiceQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type decisionRequest struct {
	Model string `json:"model"`
	State struct {
		Size              int      `json:"board_size"`
		Komi              float64  `json:"komi"`
		Board             string   `json:"board"`
		Side              string   `json:"side_to_move"`
		MoveNumber        int      `json:"move_number"`
		ConsecutivePasses int      `json:"consecutive_passes"`
		RecentMoves       []string `json:"recent_moves"`
		Rules             string   `json:"rules"`
	} `json:"state"`
	Questions map[string]choiceQuestion `json:"questions"`
}

func (o *OpenRouter) Choose(ctx context.Context, turn Turn) (Decision, error) {
	if len(turn.LegalMoves) == 0 {
		return Decision{}, errors.New("no legal moves to choose from")
	}
	criteria := make(map[string]string, len(turn.LegalMoves))
	for _, move := range turn.LegalMoves {
		criteria[move] = "Place a stone at " + move + "."
		if move == "pass" {
			criteria[move] = "Pass; two consecutive passes end the game and score the board as it stands."
		}
	}
	payload := decisionRequest{Model: turn.Model, Questions: map[string]choiceQuestion{"move": {
		Type: "choice", Criteria: criteria,
		Instructions: "Choose the strongest legal Go (Baduk/Weiqi) move for side_to_move. " +
			"Maximize your final area score, considering White's komi. Protect living groups and capture enemy groups. " +
			"Dead stones are not automatically removed at scoring: capture them before passing. " +
			"Select one of the supplied moves; coordinates label board intersections.",
	}}}
	payload.State.Size, payload.State.Komi = turn.Size, turn.Komi
	payload.State.Board, payload.State.Side = turn.Board, turn.Side
	payload.State.MoveNumber, payload.State.ConsecutivePasses = turn.MoveNumber, turn.ConsecutivePasses
	payload.State.Rules = "Tromp-Taylor: Black starts; orthogonal groups with no liberties are removed, opponent first then own. " +
		"Placements must not repeat any earlier board (positional superko). Self-capture is allowed if the resulting board is new. " +
		"Pass is always legal. Two consecutive passes end play. Score stones plus empty regions bordering only your color; add komi to White. " +
		"No dead-stone adjudication, territory scoring, or prisoner points."
	payload.State.RecentMoves = turn.RecentMoves
	body, err := json.Marshal(payload)
	if err != nil {
		return Decision{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.endpoint, bytes.NewReader(body))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Authorization", "Bearer "+o.key)
	req.Header.Set("Content-Type", "application/json")
	started := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		return Decision{}, fmt.Errorf("OpenRouter request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Provider error bodies can contain reflected request data; do not log them.
		return Decision{}, fmt.Errorf("OpenRouter HTTP %d", resp.StatusCode)
	}
	const maxBody = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return Decision{}, fmt.Errorf("read OpenRouter response: %w", err)
	}
	if len(data) > maxBody {
		return Decision{}, errors.New("OpenRouter response exceeds 1 MiB")
	}
	var result struct {
		Answers map[string]struct {
			Type       string   `json:"type"`
			Choice     string   `json:"choice"`
			Confidence *float64 `json:"confidence"`
		} `json:"answers"`
		Usage struct {
			Cost *float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return Decision{}, errors.New("invalid OpenRouter JSON response")
	}
	answer, ok := result.Answers["move"]
	if !ok || answer.Type != "choice" {
		return Decision{}, errors.New("OpenRouter response lacks a move choice")
	}
	if _, ok := criteria[answer.Choice]; !ok {
		return Decision{}, errors.New("OpenRouter selected a move outside the legal choices")
	}
	if answer.Confidence == nil || *answer.Confidence < 0 || *answer.Confidence > 1 {
		return Decision{}, errors.New("invalid or missing decision confidence")
	}
	if result.Usage.Cost == nil || *result.Usage.Cost < 0 {
		return Decision{}, errors.New("invalid or missing usage cost")
	}
	return Decision{Move: answer.Choice, Confidence: *answer.Confidence,
		CostUSD: *result.Usage.Cost, Latency: time.Since(started)}, nil
}
