package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

func init() {
	register("4a", day4a)
	register("4b", day4b)
}

func day4a(_ []string) {
	scanner := bufio.NewScanner(os.Stdin)
	var numValid int
lineLoop:
	for scanner.Scan() {
		seen := make(map[string]struct{})
		for _, field := range strings.Fields(scanner.Text()) {
			if _, ok := seen[field]; ok {
				continue lineLoop
			}
			seen[field] = struct{}{}
		}
		numValid++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(numValid)
}

func day4b(_ []string) {
	scanner := bufio.NewScanner(os.Stdin)
	var numValid int
lineLoop:
	for scanner.Scan() {
		seen := make(map[string]struct{})
		for _, field := range strings.Fields(scanner.Text()) {
			sorted := []rune(field)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
			word := string(sorted)
			if _, ok := seen[word]; ok {
				continue lineLoop
			}
			seen[word] = struct{}{}
		}
		numValid++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(numValid)
}
