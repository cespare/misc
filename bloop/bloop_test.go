package bloop

import (
	"testing"
)

func BenchmarkBaseline(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = expensive()
	}
}

func BenchmarkSink(b *testing.B) {
	var x float64
	for i := 0; i < b.N; i++ {
		x = expensive()
	}
	floatSink = x
}

func BenchmarkBLoop(b *testing.B) {
	for b.Loop() {
		expensive()
	}
}

func BenchmarkBLoopAssign(b *testing.B) {
	for b.Loop() {
		_ = expensive()
	}
}

var floatSink float64

func expensive() float64 {
	x := 1.0
	for i := range int(1e6) {
		x *= float64(i) * float64(i+1) * float64(i+2) * float64(i+3)
	}
	return x
}
