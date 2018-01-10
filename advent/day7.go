package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func init() {
	register("7", day7)
}

func day7(_ []string) {
	var progs []towerProg
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		prog, err := parseTowerProg(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}
		progs = append(progs, prog)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	hasParent := make(map[string]bool)
	for _, prog := range progs {
		for _, name := range prog.children {
			hasParent[name] = true
		}
	}

	var noParents []string
	for _, prog := range progs {
		if !hasParent[prog.name] {
			noParents = append(noParents, prog.name)
		}
	}
	if len(noParents) != 1 {
		log.Fatalf("found %d programs with no parent", len(noParents))
	}
	fmt.Println(noParents[0])

	// Construct tree.
	byName := make(map[string]*towerProg)
	for i := range progs {
		prog := &progs[i]
		byName[prog.name] = prog
	}
	var addRefs func(*towerProg)
	addRefs = func(prog *towerProg) {
		prog.refs = make([]*towerProg, len(prog.children))
		for i, child := range prog.children {
			ref1 := byName[child]
			prog.refs[i] = ref1
			addRefs(ref1)
		}
	}
	root := byName[noParents[0]]
	addRefs(root)

	var treeWeight func(*towerProg) int
	treeWeight = func(prog *towerProg) int {
		if prog.treeWeight != -1 {
			return prog.treeWeight
		}
		w := prog.weight
		for _, child := range prog.refs {
			w += treeWeight(child)
		}
		prog.treeWeight = w
		return w
	}
	// diffWeights returns the index of the odd one out, or -1 if there are
	// two differing weights, or -2 if the weights are all the same.
	diffWeights := func(weights []int) (int, int) {
		if len(weights) < 2 {
			return -2, 0
		}
		if len(weights) == 2 {
			if weights[0] == weights[1] {
				return -2, 0
			}
			return -1, 0
		}
		w0, w1, w2 := weights[0], weights[1], weights[2]
		var w int
		switch {
		case w0 == w1:
			w = w0
		case w0 == w2:
			w = w0
		case w1 == w2:
			w = w1
		default:
			panic("three different child weights")
		}
		for i, ww := range weights {
			if ww != w {
				return i, w
			}
		}
		return -2, 0
	}
	var findCorrectWeight func(*towerProg, int) (int, bool)
	findCorrectWeight = func(prog *towerProg, target int) (int, bool) {
		childWeights := make([]int, len(prog.refs))
		for i, ref := range prog.refs {
			childWeights[i] = treeWeight(ref)
		}
		var childSum int
		for _, w := range childWeights {
			childSum += w
		}
		diff, w := diffWeights(childWeights)
		if diff == -2 {
			if target < 0 {
				panic("tower seems balanced")
			}
			fixed := target - childSum
			if fixed == prog.weight {
				panic("explored definitely-OK tower")
			}
			return fixed, true
		}
		if diff == -1 {
			fixed0, imm0 := findCorrectWeight(prog.refs[0], childWeights[1])
			fixed1, imm1 := findCorrectWeight(prog.refs[1], childWeights[0])
			if imm0 && imm1 {
				panic("ambiguous pair")
			}
			if !imm0 && !imm1 {
				panic("more than one weight error")
			}
			if imm0 {
				return fixed1, false
			}
			return fixed0, false
		}
		fixed, _ := findCorrectWeight(prog.refs[diff], w)
		return fixed, false
	}
	fixed, _ := findCorrectWeight(root, -1)
	fmt.Println(fixed)
}

type towerProg struct {
	name       string
	weight     int
	treeWeight int
	children   []string
	refs       []*towerProg
}

func parseTowerProg(s string) (towerProg, error) {
	var prog towerProg
	prog.treeWeight = -1
	parts := strings.SplitN(s, "->", 2)
	selfParts := strings.Fields(parts[0])
	if len(selfParts) != 2 {
		return prog, fmt.Errorf("bad self part %q", parts[0])
	}
	prog.name = selfParts[0]
	weightStr := selfParts[1]
	if len(weightStr) < 3 || weightStr[0] != '(' || weightStr[len(weightStr)-1] != ')' {
		return prog, fmt.Errorf("bad weight part %q", weightStr)
	}
	var err error
	prog.weight, err = strconv.Atoi(weightStr[1 : len(weightStr)-1])
	if err != nil {
		return prog, err
	}

	if len(parts) > 1 {
		for _, name := range strings.Split(parts[1], ",") {
			prog.children = append(prog.children, strings.TrimSpace(name))
		}
	}
	return prog, nil
}
