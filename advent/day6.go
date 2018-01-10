package main

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
)

func init() {
	register("6", day6)
}

func day6(args []string) {
	var mem memory
	for _, arg := range args {
		n, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		mem.banks = append(mem.banks, n)
	}

	seen := map[string]struct{}{mem.String(): {}}
	var first string
	for i := 1; ; i++ {
		mem.cycle()
		s := mem.String()
		if _, ok := seen[s]; ok {
			fmt.Println(i)
			first = s
			break
		}
		seen[s] = struct{}{}
	}

	for i := 1; ; i++ {
		mem.cycle()
		if mem.String() == first {
			fmt.Println(i)
			return
		}
	}
}

type memory struct {
	banks []int64
}

func (m *memory) String() string {
	var b bytes.Buffer
	for _, numBlocks := range m.banks {
		fmt.Fprintf(&b, "%d,", numBlocks)
	}
	return b.String()
}

func (m *memory) cycle() {
	max := int64(-1)
	var j int
	for i, numBlocks := range m.banks {
		if numBlocks > max {
			j = i
			max = numBlocks
		}
	}
	m.banks[j] = 0
	for ; max > 0; max-- {
		j = (j + 1) % len(m.banks)
		m.banks[j]++
	}
}
