package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
)

// instructions is the assembled form of the following which adds two numbers.
// rdi points to the beginning of a Go function stack frame.
// (Intel syntax.)
//
// mov rax, [rdi]
// add rax, [rdi+8]
// mov [rdi+16], rax
// ret
//
const instructions = "488b074803470848894710c3"

func trampoline() // implemented in asm

// A closure is the internal representation of a Go func.
// (Actually a Go func is a pointer to one of these.)
type closure struct {
	code uintptr
	ctx  uintptr
}

// An iface is the internal representation of a Go interface.
// The second value is always a pointer (as of Go 1.5).
type iface struct {
	typ uintptr
	p   uintptr
}

func build(b []byte, f interface{}) {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Func {
		panic("build requires pointer to func")
	}

	// Synthesize a a closure that points at the trampoline but has the
	// address of the buffer code as its context.
	// (Note that trampoline loads this pointer from the known register DX.)
	tramp := trampoline // get a closure from trampoline
	c := &closure{
		code: **(**uintptr)(unsafe.Pointer(&tramp)),
		ctx:  (*(*reflect.SliceHeader)(unsafe.Pointer(&b))).Data,
	}
	f1 := *(*func())(unsafe.Pointer(&c))

	// Modify f to replace its function pointer with f1.
	fi := *(*iface)(unsafe.Pointer(&f))
	*(*func())(unsafe.Pointer(fi.p)) = f1
}

func main() {
	instr, err := hex.DecodeString(instructions)
	if err != nil {
		panic(err)
	}

	// Get mmap-ed pages for code.
	pageSize := os.Getpagesize()
	size := ((len(instr)-1)/pageSize + 1) * pageSize
	b, err := unix.Mmap(-1, 0, size, unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		log.Fatal("mmap error:", err)
	}
	defer unix.Munmap(b)
	copy(b, instr)

	// Construct a function.
	var add func(a, b int) int
	build(b, &add)

	// Make the code executable.
	if err := unix.Mprotect(b, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		log.Fatal("mprotect error:", err)
	}

	// Run the synthesized function.
	fmt.Println("add(1, 2):", add(1, 2))
	fmt.Println("add(3, -4):", add(3, -4))
}
