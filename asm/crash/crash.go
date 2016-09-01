package main

// crash is a silly function that does roughly
//   return b[len(b):len(b):len(b)]
func crash(b []byte) []byte

func main() {
	for {
		b := make([]byte, 1)
		b0 := crash(b)
		var v interface{} = b0
		_ = v
	}
}
