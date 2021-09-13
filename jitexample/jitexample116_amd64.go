//go:build amd64 && !go1.17
// +build amd64,!go1.17

package main

// instructions is the assembled form of the following which adds two numbers.
// This uses the Go ABI0 calling convention. (Intel syntax.)
//
//   mov rax, [rsp+8]
//   add rax, [rsp+16]
//   mov [rsp+24], rax
//   ret
//
const instructions = "488b 4424 0848 0344 2410 4889 4424 18c3"
