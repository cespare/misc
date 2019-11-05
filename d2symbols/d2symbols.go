// d2symbols solves a xenophage grid puzzle in Destiny 2. The 4 symbols are
// named E, T, A, M after the letters they most resemble.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

type game struct {
	start state
	goal  state
}

type state [9]byte

type point struct {
	prev state
	edge int
}

func (g *game) solve() (solution string, ok bool) {
	space := map[state]*point{g.start: nil}
	frontier := []state{g.start}
	for len(frontier) > 0 {
		st := frontier[0]
		frontier = frontier[1:]
		if st == g.goal {
			var seq []int
			// Reconstruct path.
			for p := space[st]; p != nil; p = space[p.prev] {
				seq = append(seq, p.edge)
			}
			// Reverse the path to put it in the forward order.
			for i := 0; i < len(seq)/2; i++ {
				seq[i], seq[len(seq)-i-1] = seq[len(seq)-i-1], seq[i]
			}
			// Display the values 1-indexed.
			for i, n := range seq {
				seq[i] = n + 1
			}
			return fmt.Sprintf("%v", seq), true
		}
		for edge, next := range nextStates(st) {
			if _, ok := space[next]; ok {
				continue // already explored
			}
			space[next] = &point{
				prev: st,
				edge: edge,
			}
			frontier = append(frontier, next)
		}
	}
	return "", false
}

func nextStates(st state) [9]state {
	var states [9]state
	states[0] = advanceState(st, 0, 1, 2, 3, 6)
	states[1] = advanceState(st, 0, 1, 2, 4, 7)
	states[2] = advanceState(st, 0, 1, 2, 5, 8)
	states[3] = advanceState(st, 0, 3, 4, 5, 6)
	states[4] = advanceState(st, 1, 3, 4, 5, 7)
	states[5] = advanceState(st, 2, 3, 4, 5, 8)
	states[6] = advanceState(st, 0, 3, 6, 7, 8)
	states[7] = advanceState(st, 1, 4, 6, 7, 8)
	states[8] = advanceState(st, 2, 5, 6, 7, 8)
	return states
}

func advanceState(st state, indexes ...int) state {
	for _, i := range indexes {
		st[i] = nextSymbol(st[i])
	}
	return st
}

func nextSymbol(sym byte) byte {
	switch sym {
	case 'E':
		return 'T'
	case 'T':
		return 'A'
	case 'A':
		return 'M'
	case 'M':
		return 'E'
	default:
		panic("bad symbol")
	}
}

func main() {
	log.SetFlags(0)
	if len(os.Args) != 2 {
		log.Fatal("Usage: `d2symbols 'T:AMAEMEMTM'` (for example)")
	}

	g, err := parseInput(os.Args[1])
	if err != nil {
		log.Fatalln("Bad input:", err)
	}
	solution, ok := g.solve()
	if !ok {
		log.Fatal("no solution")
	}
	log.Println(solution)
}

func parseInput(s string) (*game, error) {
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("no : in input")
	}
	if len(parts[0]) != 1 {
		return nil, fmt.Errorf("goal must be a single byte; got %q", parts[0])
	}
	if !checkSymbol(parts[0][0]) {
		return nil, fmt.Errorf("bad goal symbol: %q", parts[0][0])
	}
	if len(parts[1]) != 9 {
		return nil, fmt.Errorf("state must have 9 symbols; got %q", parts[1])
	}
	for i := 0; i < 9; i++ {
		if !checkSymbol(parts[1][i]) {
			return nil, fmt.Errorf("bad state symbol: %q", parts[1][i])
		}
	}
	var g game
	copy(g.start[:], parts[1])
	for i := range g.goal {
		g.goal[i] = parts[0][0]
	}
	return &g, nil
}

func checkSymbol(b byte) bool {
	switch b {
	case 'E', 'T', 'A', 'M':
		return true
	default:
		return false
	}
}
