// +build ignore

package main

//go:noinline
func gt(i, j int64) bool {
	return i > j
}

var sink int64
var sinkb bool

func main() {
	sinkb = gt(1, 2)
}
