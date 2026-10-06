package badukmatch

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const columns = "ABCDEFGHJKLMNOPQRST"

// position uses immutable strings and copies history when adding a board.
type position struct {
	size    int
	cells   string // Top row first; X is Black, O is White, . is empty.
	black   bool
	passes  int
	history []string
}

func newPosition(size int) position {
	cells := strings.Repeat(".", size*size)
	return position{size: size, cells: cells, black: true, history: []string{cells}}
}

func (p position) side() string {
	if p.black {
		return "black"
	}
	return "white"
}

func coordinate(index, size int) string {
	return fmt.Sprintf("%c%d", columns[index%size], size-index/size)
}

func point(move string, size int) (int, error) {
	if len(move) < 2 {
		return 0, errors.New("invalid coordinate")
	}
	x := strings.IndexByte(columns[:size], move[0])
	row, err := strconv.Atoi(move[1:])
	if err != nil || x < 0 || row < 1 || row > size {
		return 0, errors.New("coordinate outside board")
	}
	index := (size-row)*size + x
	if coordinate(index, size) != move {
		return 0, errors.New("coordinate must use uppercase letters and canonical row numbers")
	}
	return index, nil
}

func neighbors(index, size int) []int {
	points := make([]int, 0, 4)
	x, y := index%size, index/size
	if x > 0 {
		points = append(points, index-1)
	}
	if x+1 < size {
		points = append(points, index+1)
	}
	if y > 0 {
		points = append(points, index-size)
	}
	if y+1 < size {
		points = append(points, index+size)
	}
	return points
}

func group(cells []byte, start, size int) ([]int, bool) {
	stones := []int{start}
	seen := make([]bool, len(cells))
	seen[start] = true
	liberty := false
	for i := 0; i < len(stones); i++ {
		for _, next := range neighbors(stones[i], size) {
			if cells[next] == '.' {
				liberty = true
			}
			if !seen[next] && cells[next] == cells[start] {
				seen[next] = true
				stones = append(stones, next)
			}
		}
	}
	return stones, liberty
}

// play implements the core Tromp-Taylor rules, including self-capture and PSK.
func (p position) play(move string) (position, error) {
	if p.passes >= 2 {
		return p, errors.New("game has ended")
	}
	if move == "pass" {
		p.black = !p.black
		p.passes++
		return p, nil
	}
	index, err := point(move, p.size)
	if err != nil {
		return p, err
	}
	if p.cells[index] != '.' {
		return p, errors.New("intersection is occupied")
	}
	cells := []byte(p.cells)
	own, opponent := byte('X'), byte('O')
	if !p.black {
		own, opponent = opponent, own
	}
	cells[index] = own
	for _, next := range neighbors(index, p.size) {
		if cells[next] != opponent {
			continue
		}
		stones, liberty := group(cells, next, p.size)
		if !liberty {
			for _, stone := range stones {
				cells[stone] = '.'
			}
		}
	}
	stones, liberty := group(cells, index, p.size)
	if !liberty {
		for _, stone := range stones {
			cells[stone] = '.'
		}
	}
	board := string(cells)
	if slices.Contains(p.history, board) {
		return p, errors.New("positional superko: move repeats an earlier board")
	}
	p.cells, p.black, p.passes = board, !p.black, 0
	p.history = append(slices.Clone(p.history), board)
	return p, nil
}

func (p position) legalMoves() []string {
	if p.passes >= 2 {
		return nil
	}
	moves := make([]string, 0, p.size*p.size+1)
	for index := range p.cells {
		if p.cells[index] != '.' {
			continue
		}
		move := coordinate(index, p.size)
		if _, err := p.play(move); err == nil {
			moves = append(moves, move)
		}
	}
	return append(moves, "pass")
}

func (p position) draw() string {
	var out strings.Builder
	out.WriteString("X = black, O = white, . = empty\n   ")
	for x := 0; x < p.size; x++ {
		fmt.Fprintf(&out, "%c ", columns[x])
	}
	out.WriteByte('\n')
	for y := 0; y < p.size; y++ {
		fmt.Fprintf(&out, "%2d ", p.size-y)
		for x := 0; x < p.size; x++ {
			fmt.Fprintf(&out, "%c ", p.cells[y*p.size+x])
		}
		out.WriteByte('\n')
	}
	return out.String()
}

type Score struct {
	BlackArea int     `json:"black_area"`
	WhiteArea int     `json:"white_area"`
	Komi      float64 `json:"komi"`
}

func (s Score) result() string {
	difference := float64(s.BlackArea-s.WhiteArea) - s.Komi
	if difference == 0 {
		return "0"
	}
	winner := "B+"
	if difference < 0 {
		winner, difference = "W+", -difference
	}
	return winner + strconv.FormatFloat(difference, 'f', -1, 64)
}

func (p position) score(komi float64) Score {
	score := Score{Komi: komi}
	seen := make([]bool, len(p.cells))
	for index := range p.cells {
		switch p.cells[index] {
		case 'X':
			score.BlackArea++
		case 'O':
			score.WhiteArea++
		case '.':
			if seen[index] {
				continue
			}
			region := []int{index}
			seen[index] = true
			black, white := false, false
			for i := 0; i < len(region); i++ {
				for _, next := range neighbors(region[i], p.size) {
					switch p.cells[next] {
					case 'X':
						black = true
					case 'O':
						white = true
					case '.':
						if !seen[next] {
							seen[next] = true
							region = append(region, next)
						}
					}
				}
			}
			if black && !white {
				score.BlackArea += len(region)
			}
			if white && !black {
				score.WhiteArea += len(region)
			}
		}
	}
	return score
}
