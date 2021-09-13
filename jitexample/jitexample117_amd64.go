//go:build amd64 && go1.17
// +build amd64,go1.17

package main

// instructions is the assembled form of the following which adds two numbers.
// This uses the Go ABIInternal calling convention as of Go 1.17.
// (Intel syntax.)
//
//   add rax, rbx
//   ret
//
const instructions = "4801 d8c3"
