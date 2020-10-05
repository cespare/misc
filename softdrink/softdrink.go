package main

import (
	"fmt"
	"log"
	"strings"
)

var drinks = [][]byte{
	[]byte("RINL"),
	[]byte("KPSA"),
	[]byte("YFTD"),
	[]byte("EOCG"),
}

var goalNamed = map[byte]int{
	'R': 1,
	'I': 2,
	'L': 2,
	'K': 1,
	'P': 2,
	'S': 1,
	'A': 2,
	'Y': 2,
	'F': 3,
	'T': 2,
	'D': 1,
	'E': 1,
	'O': 2,
	'C': 1,
	'G': 2,
}

var goal = make(map[pos]int)

func init() {
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			d := drinks[r][c]
			goal[pos{r, c}] = goalNamed[d]
		}
	}
}

type pos struct {
	row int
	col int
}

type state struct {
	hist []pos
	disp map[pos]int
}

func (s *state) solve() []pos {
	solved := true
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			p := pos{r, c}
			have := s.disp[p]
			want := goal[p]
			switch {
			case have < want:
				solved = false
			case have > want:
				return nil
			}
		}
	}
	if solved {
		return s.hist
	}
	if len(s.hist) == 5 {
		return nil
	}
	prev := pos{-1, -1}
	if len(s.hist) > 0 {
		prev = s.hist[len(s.hist)-1]
	}
	for r := 0; r < 4; r++ {
	rowLoop:
		for c := 0; c < 4; c++ {
			p := pos{r, c}
			for _, p0 := range s.hist {
				if p0 == p {
					continue rowLoop
				}
			}

			// Apply changes.
			for r0 := 0; r0 < 4; r0++ {
				p0 := pos{r0, c}
				if p0.row == prev.row || p0.col == prev.col {
					continue
				}
				s.disp[p0]++
			}
			for c0 := 0; c0 < 4; c0++ {
				p0 := pos{r, c0}
				if p0 == p {
					continue
				}
				if p0.row == prev.row || p0.col == prev.col {
					continue
				}
				s.disp[p0]++
			}
			s.hist = append(s.hist, p)

			if sol := s.solve(); sol != nil {
				return sol
			}

			// Didn't work -- undo changes.
			for r0 := 0; r0 < 4; r0++ {
				p0 := pos{r0, c}
				if p0.row == prev.row || p0.col == prev.col {
					continue
				}
				s.disp[p0]--
			}
			for c0 := 0; c0 < 4; c0++ {
				p0 := pos{r, c0}
				if p0 == p {
					continue
				}
				if p0.row == prev.row || p0.col == prev.col {
					continue
				}
				s.disp[p0]--
			}
			s.hist = s.hist[:len(s.hist)-1]
		}
	}
	return nil
}

func solutionString(sol []pos) string {
	var b strings.Builder
	for _, p := range sol {
		b.WriteByte(drinks[p.row][p.col])
	}
	return b.String()
}

func main() {
	s := &state{disp: make(map[pos]int)}
	sol := s.solve()
	if sol == nil {
		log.Fatal("No solution")
	}
	fmt.Println(solutionString(sol))
}
