package main

var charSet [4]uint64

func inCharSet(b byte) bool {
	return charSet[b>>6]&(1<<(b&63)) != 0
}

func init() {
	chars := "qz:}t"
	for i := 0; i < len(chars); i++ {
		b := chars[i]
		charSet[b>>6] |= 1 << (b & 63)
	}
}
