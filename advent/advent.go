package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		var names []string
		for name := range solutions {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool { return nameLess(names[i], names[j]) })
		fmt.Fprintf(os.Stderr, "usage: %s [solution] [args...]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "where solution is one of:")
		for _, name := range names {
			fmt.Fprintln(os.Stderr, name)
		}
		os.Exit(1)
	}

	fn, ok := solutions[os.Args[1]]
	if !ok {
		log.Fatalf("unknown solution %q", os.Args[1])
	}
	fn(os.Args[2:])
}

var solutions = make(map[string]func([]string))

func register(name string, fn func([]string)) {
	if _, ok := solutions[name]; ok {
		panic(fmt.Sprintf("duplicate solutions registered for %q", name))
	}
	solutions[name] = fn
}

func nameLess(name0, name1 string) bool {
	n0, s0 := splitName(name0)
	n1, s1 := splitName(name1)
	if n0 < n1 {
		return true
	}
	if n0 > n1 {
		return false
	}
	return s0 < s1
}

func splitName(name string) (int, string) {
	i := 0
	for ; i < len(name); i++ {
		c := name[i]
		if c < '0' || c > '9' {
			break
		}
	}
	n, err := strconv.Atoi(name[:i])
	if err != nil {
		panic(err)
	}
	return n, name[i:]
}
