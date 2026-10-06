package badukmatch

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func diagram(rows ...string) position {
	cells := strings.Join(rows, "")
	return position{size: len(rows), cells: cells, black: true, history: []string{cells}}
}

func TestInitialChoicesAndCoordinates(t *testing.T) {
	for _, size := range []int{9, 13, 19} {
		p := newPosition(size)
		moves := p.legalMoves()
		require.Len(t, moves, size*size+1)
		require.Equal(t, "pass", moves[len(moves)-1])
		for index := range p.cells {
			decoded, err := point(coordinate(index, size), size)
			require.NoError(t, err)
			require.Equal(t, index, decoded)
		}
	}
	for _, bad := range []string{"I1", "A0", "A10", "a1", "A01", "A+1", "Z1", "A", ""} {
		_, err := newPosition(9).play(bad)
		require.Error(t, err, bad)
	}
}

func TestCaptureRemovesAllAdjacentGroupsBeforeCheckingOwnLiberties(t *testing.T) {
	p := diagram("..X..", ".XOX.", "XO.OX", ".XOX.", "..X..")
	next, err := p.play("C3")
	require.NoError(t, err)
	require.Zero(t, strings.Count(next.cells, "O"))
	require.Equal(t, 9, strings.Count(next.cells, "X"))
	require.Equal(t, 4, strings.Count(p.cells, "O"))
	require.Len(t, p.history, 1)
	_, err = next.play("C3")
	require.ErrorContains(t, err, "occupied")
}

func TestTrompTaylorSelfCaptureAndNoOpSuicide(t *testing.T) {
	p := diagram(".O.", "OXO", "O.O")
	next, err := p.play("B1")
	require.NoError(t, err)
	require.Equal(t, ".O.O.OO.O", next.cells)
	require.Contains(t, p.legalMoves(), "B1")
	_, err = p.play("A3")
	require.ErrorContains(t, err, "superko")
	require.NotContains(t, p.legalMoves(), "A3")
}

func TestKoAndOlderBoardRepetition(t *testing.T) {
	p := diagram(".....", "..XO.", ".XO.O", "..XO.", ".....")
	next, err := p.play("D3")
	require.NoError(t, err)
	_, err = next.play("C3")
	require.ErrorContains(t, err, "superko")
	require.NotContains(t, next.legalMoves(), "C3")
	// Retain a forbidden board earlier than the immediately previous position.
	next.history = []string{p.cells, newPosition(5).cells, next.cells}
	_, err = next.play("C3")
	require.ErrorContains(t, err, "superko")
	passed, err := next.play("pass")
	require.NoError(t, err)
	require.Equal(t, next.history, passed.history)
}

func TestPassesResetOnlyAfterAPlacement(t *testing.T) {
	p := newPosition(9)
	var err error
	p, err = p.play("pass")
	require.NoError(t, err)
	require.Equal(t, 1, p.passes)
	p, err = p.play("D4")
	require.NoError(t, err)
	require.Zero(t, p.passes)
	p, err = p.play("pass")
	require.NoError(t, err)
	p, err = p.play("pass")
	require.NoError(t, err)
	require.Empty(t, p.legalMoves())
	_, err = p.play("A1")
	require.ErrorContains(t, err, "ended")
}

func TestAreaScoringAndKomi(t *testing.T) {
	p := diagram("XXXOO", "X.XO.", "XXXOO", ".....", ".....")
	score := p.score(2.5)
	require.Equal(t, 9, score.BlackArea)
	require.Equal(t, 6, score.WhiteArea)
	require.Equal(t, "B+0.5", score.result())
	require.Equal(t, "W+4.5", p.score(7.5).result())
	require.Equal(t, "0", p.score(3).result())
	require.Equal(t, Score{Komi: 7.5}, newPosition(9).score(7.5))
}

func TestIndependentSgfmillFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/rules.json")
	require.NoError(t, err)
	var fixture struct {
		Cases []struct {
			Size  int
			Steps []struct {
				Before         string
				Side           string
				Choices        []string
				Move           string
				After          string
				AreaDifference int `json:"area_difference"`
			}
		}
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	for _, tc := range fixture.Cases {
		p := newPosition(tc.Size)
		for n, step := range tc.Steps {
			require.Equal(t, step.Before, p.cells, "size=%d step=%d", tc.Size, n)
			require.Equal(t, step.Side, p.side())
			require.Equal(t, step.Choices, p.legalMoves(), "size=%d step=%d", tc.Size, n)
			p, err = p.play(step.Move)
			require.NoError(t, err)
			require.Equal(t, step.After, p.cells)
			score := p.score(0)
			require.Equal(t, step.AreaDifference, score.BlackArea-score.WhiteArea)
		}
	}
}
