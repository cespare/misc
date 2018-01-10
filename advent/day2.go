package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func init() {
	register("2a", day2a)
	register("2b", day2b)
}

func day2a(_ []string) {
	mat, err := parseMatrix(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	var checksum int64
	for _, row := range mat {
		if len(row) == 0 {
			log.Fatal("empty row")
		}
		min, max := row[0], row[0]
		for _, n := range row {
			if n < min {
				min = n
			}
			if n > max {
				max = n
			}
		}
		checksum += max - min
	}
	fmt.Println(checksum)
}

func day2b(_ []string) {
	mat, err := parseMatrix(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	var sum int64
rowLoop:
	for _, row := range mat {
		for i := 0; i < len(row); i++ {
			for j := i + 1; j < len(row); j++ {
				n0, n1 := row[i], row[j]
				if n0 > n1 {
					n0, n1 = n1, n0
				}
				if n1%n0 == 0 {
					sum += (n1 / n0)
					continue rowLoop
				}
			}
		}
	}
	fmt.Println(sum)
}

func parseMatrix(r io.Reader) ([][]int64, error) {
	var mat [][]int64
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		row := make([]int64, len(fields))
		for i, field := range fields {
			n, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				return nil, err
			}
			row[i] = n
		}
		mat = append(mat, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mat, nil
}
