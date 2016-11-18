package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

const (
	N = 65e3
)

type Columns struct {
	A []uint16
	B []uint16
	C []uint16
	D []uint16
}

type Chunks []Columns

func NewRandomChunks(n int) Chunks {
	chunks := make(Chunks, n)
	for i := range chunks {
		chunks[i] = NewRandomColumns()
	}
	return chunks
}

func NewRandomColumns() Columns {
	return Columns{
		A: randomVals(N),
		B: randomVals(N),
		C: randomVals(N),
		D: randomVals(N),
	}
}

func randomVals(n int) []uint16 {
	s := make([]uint16, n)
	for i := range s {
		s[i] = uint16(rand.Int63n(int64(math.MaxUint16)))
	}
	return s
}

func (c Chunks) ScanLockstep(x uint16, result, selectedRows *int64) func(b *testing.B) {
	return func(b *testing.B) {
		var sum int64
		var selected int64
		for i := 0; i < b.N; i++ {
			sum = 0
			selected = 0
			for i, v := range c.A {
				if v < x {
					sum += int64(c.B[i]) + int64(c.C[i]) + int64(c.D[i])
					selected++
				}
			}
		}
		*result = sum
		*selectedRows = selected
	}
}

func (c *Columns) ScanSeparately(x uint16, result, selectedRows *int64) func(b *testing.B) {
	return func(b *testing.B) {
		var sum int64
		var plist []uint16
		for i := 0; i < b.N; i++ {
			sum = 0
			plist = plist[:0]
			for i, v := range c.A {
				if v < x {
					plist = append(plist, uint16(i))
				}
			}
			for _, i := range plist {
				sum += int64(c.B[int(i)])
			}
			for _, i := range plist {
				sum += int64(c.C[int(i)])
			}
			for _, i := range plist {
				sum += int64(c.D[int(i)])
			}
		}
		*result = sum
		*selectedRows = int64(len(plist))
	}
}

func (c *Columns) Bench(strategy string, x uint16) {
	name := fmt.Sprintf(
		"Query: SELECT SUM(B)+SUM(C)+SUM(D) WHERE A < %d; strategy: %s:",
		x, strategy,
	)
	var selectedRows, result int64
	switch strategy {
	case "scan separately":
		printBenchResult(name,
			testing.Benchmark(c.ScanSeparately(x, &result, &selectedRows)))
	case "scan in lockstep":
		printBenchResult(name,
			testing.Benchmark(c.ScanLockstep(x, &result, &selectedRows)))
	default:
		panic("bad strategy")
	}
	fmt.Printf("rows selected: %d (%g%%); result: %d\n",
		selectedRows, float64(selectedRows)/float64(N)*100, result)
}

func printBenchResult(name string, br testing.BenchmarkResult) {
	fmt.Printf("%s %s/op\n", name, time.Duration(br.NsPerOp()))
}

func main() {
	chunks := NewRandomChunks()
	chunks.Bench("scan separately", 100)
	chunks.Bench("scan in lockstep", 100)
	chunks.Bench("scan separately", 10000)
	chunks.Bench("scan in lockstep", 10000)
	chunks.Bench("scan separately", math.MaxUint16)
	chunks.Bench("scan in lockstep", math.MaxUint16)
}
