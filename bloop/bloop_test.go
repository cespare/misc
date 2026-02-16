package bloop

import "testing"

func foo() int {
	x := 123
	for i := range 1000 {
		x *= i
	}
	return x
}

func BenchmarkBasic(b *testing.B) {
	for range b.N {
		foo()
	}
}

func BenchmarkSub(b *testing.B) {
	b.Run("a", func(b *testing.B) {
		for range b.N {
			foo()
		}
	})
}
