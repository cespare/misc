package main

import "reflect"

// crash is a silly function that does
//   return b[len(b):len(b):len(b)]
func crash(b []byte) []byte

var x byte

func main() {
	for {
		b := make([]byte, 32)
		b0 := crash(b)
		if !reflect.DeepEqual(b0, b0) {
			panic("first")
		}
	}
}
