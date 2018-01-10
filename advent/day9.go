package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

func init() {
	register("9", day9)
}

func day9(_ []string) {
	g, err := parseStream(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(g.score(1))
	fmt.Println(g.count())
}

type parseState int

const (
	stateOuter parseState = iota
	stateClosed
	stateGarbage
	stateEscaped
)

type group struct {
	children []interface{} // *group or string (garbage)
}

func (g *group) score(n int) int {
	score := n
	for _, child := range g.children {
		if g1, ok := child.(*group); ok {
			score += g1.score(n + 1)
		}
	}
	return score
}

func (g *group) count() int {
	var count int
	for _, child := range g.children {
		switch child := child.(type) {
		case *group:
			count += child.count()
		case string:
			count += len(child)
		}
	}
	return count
}

func parseStream(r io.Reader) (*group, error) {
	br := bufio.NewReader(r)
	state := stateOuter
	var stack []*group
	var g *group
	var garbage []byte
	complete := false
	for i := 0; ; i++ {
		c, err := br.ReadByte()
		if err == io.EOF {
			if !complete {
				return nil, errors.New("unexpected EOF")
			}
			if g == nil {
				return nil, errors.New("empty stream")
			}
			return g, nil
		}
		if err != nil {
			return nil, err
		}
		if c == '\n' {
			continue
		}
		if complete {
			return nil, errors.New("stream must contain a single group")
		}
		switch state {
		case stateOuter:
			switch c {
			case '{':
				if g != nil {
					stack = append(stack, g)
				}
				g = new(group)
			case '<':
				if g == nil {
					return nil, errors.New("stream must be a group")
				}
				state = stateGarbage
			case '}':
				if g == nil {
					return nil, errors.New("stream starts with }")
				}
				if len(stack) == 0 {
					complete = true
				} else {
					parent := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					parent.children = append(parent.children, g)
					g = parent
				}
				state = stateClosed
			default:
				return nil, fmt.Errorf("unexpected %q at pos %d", c, i)
			}
		case stateClosed:
			switch c {
			case ',':
				state = stateOuter
			case '}':
				if len(stack) == 0 {
					complete = true
				} else {
					parent := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					parent.children = append(parent.children, g)
					g = parent
				}
			default:
				return nil, fmt.Errorf("unexpected %q at pos %d", c, i)
			}
		case stateGarbage:
			switch c {
			case '!':
				state = stateEscaped
			case '>':
				g.children = append(g.children, string(garbage))
				garbage = nil
				state = stateClosed
			default:
				garbage = append(garbage, c)
			}
		case stateEscaped:
			state = stateGarbage
		}
	}
}
