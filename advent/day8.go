package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func init() {
	register("8", day8)
}

func day8(_ []string) {
	var insns []instruction
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		insn, ok := parseInstruction(scanner.Text())
		if !ok {
			log.Fatalf("bad instruction %q", insn)
		}
		insns = append(insns, insn)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	c := newCPU()
	for _, insn := range insns {
		c.run(insn)
	}
	var max int64
	for _, v := range c.regs {
		if v > max {
			max = v
		}
	}
	fmt.Println(max)
	fmt.Println(c.max)
}

type instruction struct {
	reg   string
	delta int64
	cond  struct {
		reg string
		op  string
		val int64
	}
}

func parseInstruction(s string) (instruction, bool) {
	var insn instruction
	parts := strings.Fields(s)
	if len(parts) != 7 {
		return insn, false
	}
	insn.reg = parts[0]
	var err error
	insn.delta, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return insn, false
	}
	switch parts[1] {
	case "inc":
	case "dec":
		insn.delta = -insn.delta
	default:
		return insn, false
	}
	if parts[3] != "if" {
		return insn, false
	}
	insn.cond.reg = parts[4]
	insn.cond.op = parts[5]
	switch insn.cond.op {
	case "==", "!=", "<", "<=", ">", ">=":
	default:
		return insn, false
	}
	insn.cond.val, err = strconv.ParseInt(parts[6], 10, 64)
	if err != nil {
		return insn, false
	}
	return insn, true
}

type cpu struct {
	regs map[string]int64
	max  int64
}

func newCPU() *cpu {
	return &cpu{regs: make(map[string]int64)}
}

func (c *cpu) run(insn instruction) {
	cv, ok := c.regs[insn.cond.reg]
	if !ok {
		c.regs[insn.cond.reg] = 0
	}
	var cond bool
	switch insn.cond.op {
	case "==":
		cond = cv == insn.cond.val
	case "!=":
		cond = cv != insn.cond.val
	case "<":
		cond = cv < insn.cond.val
	case "<=":
		cond = cv <= insn.cond.val
	case ">":
		cond = cv > insn.cond.val
	case ">=":
		cond = cv >= insn.cond.val
	}
	if cond {
		old := c.regs[insn.reg]
		v := old + insn.delta
		if v > c.max {
			c.max = v
		}
		c.regs[insn.reg] = v
	}
}
