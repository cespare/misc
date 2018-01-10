package main

import (
	"fmt"
	"log"
	"strconv"
)

func init() {
	register("3a", day3a)
	register("3b", day3b)
}

func day3a(args []string) {
	if len(args) != 1 {
		log.Fatal("need 1 arg")
	}
	n, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	x, y := spiralCoords(n)
	var d int64
	if x > 0 {
		d += x
	} else {
		d -= x
	}
	if y > 0 {
		d += y
	} else {
		d -= y
	}
	fmt.Println(d)
}

func day3b(args []string) {
	if len(args) != 1 {
		log.Fatal("need 1 arg")
	}
	n, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	m := map[int64]int64{1: 1}
	for i := int64(2); ; i++ {
		var sum int64
		for _, nb := range neighbors(i) {
			sum += m[nb]
		}
		if sum > n {
			fmt.Println(sum)
			return
		}
		m[i] = sum
	}
}

func neighbors(n int64) []int64 {
	x, y := spiralCoords(n)
	return []int64{
		spiralIndex(x+1, y),
		spiralIndex(x+1, y+1),
		spiralIndex(x, y+1),
		spiralIndex(x-1, y+1),
		spiralIndex(x-1, y),
		spiralIndex(x-1, y-1),
		spiralIndex(x, y-1),
		spiralIndex(x+1, y-1),
	}
}

var globalSpiralState = newSpiralState()

type spiralState struct {
	indexToCoord map[int64]vec2
	coordToIndex map[vec2]int64
	n            int64
	idir         int
	d0           int64
	d            int64
	par          bool
	v            vec2
}

func newSpiralState() *spiralState {
	return &spiralState{
		indexToCoord: map[int64]vec2{1: {0, 0}},
		coordToIndex: map[vec2]int64{{0, 0}: 1},
		n:            1,
		d0:           1,
		d:            1,
	}
}

var spiralDirs = []vec2{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

func (ss *spiralState) advance() vec2 {
	if ss.d == 0 { // turn
		ss.idir = (ss.idir + 1) % len(spiralDirs)
		if ss.par {
			ss.d0++
		}
		ss.par = !ss.par
		ss.d = ss.d0
	}
	ss.v = ss.v.add(spiralDirs[ss.idir])
	ss.d--
	ss.n++
	ss.indexToCoord[ss.n] = ss.v
	ss.coordToIndex[ss.v] = ss.n
	return ss.v
}

func spiralCoords(n int64) (x, y int64) {
	if n < 1 {
		panic("input must be >= 1")
	}
	for globalSpiralState.n < n {
		globalSpiralState.advance()
	}
	v := globalSpiralState.indexToCoord[n]
	return v.x, v.y
}

func spiralIndex(x, y int64) int64 {
	v := vec2{x, y}
	n, ok := globalSpiralState.coordToIndex[v]
	if ok {
		return n
	}
	for {
		v1 := globalSpiralState.advance()
		if v1 == v {
			break
		}
	}
	return globalSpiralState.coordToIndex[v]
}

type vec2 struct {
	x, y int64
}

func (v vec2) add(v1 vec2) vec2 {
	return vec2{v.x + v1.x, v.y + v1.y}
}

func (v vec2) scalarMul(n int64) vec2 {
	return vec2{v.x * n, v.y * n}
}
