package main

import (
	"fmt"
	"log"
	"math/big"
)

func init() {
	register("1a", day1a)
	register("1b", day1b)
}

func day1a(args []string) {
	if len(args) != 1 {
		log.Fatal("need 1 arg")
	}
	digits := args[0]

	var sum big.Int
	for i := 0; i < len(digits); i++ {
		j := (i + 1) % len(digits)
		if i == len(digits)-1 {
			j = 0
		}
		c := digits[i]
		if c < '0' || c > '9' {
			log.Fatalf("input contained non-digit %s", c)
		}
		if c == digits[j] {
			sum.Add(&sum, big.NewInt(int64(c)-int64('0')))
		}
	}
	fmt.Println(sum.String())
}

func day1b(args []string) {
	if len(args) != 1 {
		log.Fatal("need 1 arg")
	}
	digits := args[0]
	if len(digits)%2 != 0 {
		log.Fatal("need even number of digits")
	}

	var sum big.Int
	for i := 0; i < len(digits); i++ {
		j := (i + len(digits)/2) % len(digits)
		c := digits[i]
		if c < '0' || c > '9' {
			log.Fatalf("input contained non-digit %s", c)
		}
		if c == digits[j] {
			sum.Add(&sum, big.NewInt(int64(c)-int64('0')))
		}
	}
	fmt.Println(sum.String())
}
