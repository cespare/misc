package main

import (
	"testing"
)

func BenchmarkInline(b *testing.B) {
	const m = 3
	fn := makeFunc(m)
	runBench(b, m, fn)
}

func runBench(b *testing.B, m int, fn func([]val, []val)) {
	v0 := make([]val, m)
	v1 := make([]val, m)
	for i := 0; i < b.N; i++ {
		fn(v0, v1)
	}
}

type val int64

func (p *val) add(v val) {
	*p += v
}

func makeFunc(m int) func([]val, []val) {
	s := make([]int, m)
	return func(v0, v1 []val) {
		// Writing this as 'for i := 0; i < m; i++' makes 1.17 behave
		// the same as 1.19.
		for i := range s {
			v0[i].add(v1[i])
		}
	}
}
