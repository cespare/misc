//go:build !linux

package main

func foo() {
	panic("unimplemented")
}

func bar() {
	panic("unimplemented")
}
