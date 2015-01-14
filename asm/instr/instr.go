package instr

func Add(n, m int64) int64 {
	return n + m
}

func Add2(n, m int64) int64

// BSF returns the index of the least significant set bit,
// or -1 if the input contains no set bits.
func BSF(n int64) int
