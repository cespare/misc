package main

import (
	"github.com/cespare/misc/asm/instr"
)

func main() {
	//s := instr.Add2(5, 10)
	//s := instr.BSF32(12)
	ns := []int64{2, 3, 4}
	s := instr.Sum(ns)
	s2 := instr.Sum2(ns)
	s3 := instr.Add(3, 4)
	s4 := instr.Add2(3, 4)
	println(s)
	println(s2)
	println(s3)
	println(s4)
}
