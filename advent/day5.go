package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

func init() {
	register("5a", day5a)
	register("5b", day5b)
}

func day5a(_ []string) {
	scanner := bufio.NewScanner(os.Stdin)
	var input []int64
	for scanner.Scan() {
		n, err := strconv.ParseInt(scanner.Text(), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		input = append(input, n)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	m := machine1{insns: input}
	i := 1
	for m.step() {
		i++
	}
	fmt.Println(i)
}

type machine1 struct {
	insns []int64
	pc    int64
}

func (m *machine1) step() bool {
	next := m.pc + m.insns[m.pc]
	if next < 0 || next >= int64(len(m.insns)) {
		return false
	}
	m.insns[m.pc]++
	m.pc = next
	return true
}

func day5b(_ []string) {
	scanner := bufio.NewScanner(os.Stdin)
	var input []int64
	for scanner.Scan() {
		n, err := strconv.ParseInt(scanner.Text(), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		input = append(input, n)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	m := machine2{insns: input}
	i := 1
	for m.step() {
		i++
	}
	fmt.Println(i)
}

type machine2 struct {
	insns []int64
	pc    int64
}

func (m *machine2) step() bool {
	off := m.insns[m.pc]
	next := m.pc + off
	if next < 0 || next >= int64(len(m.insns)) {
		return false
	}
	if off >= 3 {
		off--
	} else {
		off++
	}
	m.insns[m.pc] = off
	m.pc = next
	return true
}
