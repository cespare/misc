package main

import (
	"testing"
)

func FuzzXyz(f *testing.F) {
	f.Add(0.0)
	f.Fuzz(func(t *testing.T, n int) {
		t.Fatal("blah")
	})
}
