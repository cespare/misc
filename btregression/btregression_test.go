package main

import "testing"

func BenchmarkInCharSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		const s = "yellow submarine"
		for j := 0; j < len(s); j++ {
			if inCharSet(s[j]) {
				b.Fatal("blah")
			}
		}
	}
}
