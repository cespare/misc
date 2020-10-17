package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/bits"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"
)

const gameSize = 7

func main() {
	sortByScore := flag.Bool("s", false, "Sort by score (rather than word frequency)")
	flag.Parse()

	p, err := loadPuzzle(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	corpus, err := loadCorpus("words.txt")
	if err != nil {
		log.Fatal(err)
	}

	type solution struct {
		word      string
		pangram   bool
		freqScore int
	}
	var solutions []solution
	for _, group := range corpus {
		if !p.matches(group.letters) {
			continue
		}
		pangram := bits.OnesCount32(uint32(group.letters)) == gameSize
		for _, ws := range group.words {
			solutions = append(solutions, solution{
				word:      ws.word,
				pangram:   pangram,
				freqScore: ws.score,
			})
		}
	}
	sort.Slice(solutions, func(i, j int) bool {
		s0, s1 := solutions[i], solutions[j]
		if s0.pangram != s1.pangram {
			return s0.pangram
		}
		if *sortByScore && len(s0.word) != len(s1.word) {
			return len(s0.word) > len(s1.word)
		}
		return s0.freqScore < s1.freqScore
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer tw.Flush()
	var totalScore int
	for _, sol := range solutions {
		var p string
		score := len(sol.word) - 3
		if sol.pangram {
			p = "*"
			score += 7
		}
		totalScore += score
		fmt.Fprintf(tw, "%s\t%d%s\t\n", sol.word, score, p)
	}
	fmt.Fprintf(tw, "\t%d\t\n", totalScore)
}

type letterSet uint32

func makeLetterSet(word string) letterSet {
	var set letterSet
	for i := 0; i < len(word); i++ {
		n := word[i] - 'A'
		set |= 1 << n
	}
	return set
}

type puzzle struct {
	center letterSet
	all    letterSet
}

func loadPuzzle(args []string) (puzzle, error) {
	var p puzzle
	var letters []byte
	for _, arg := range args {
		for _, r := range arg {
			switch {
			case r >= 'a' && r <= 'z':
				letters = append(letters, byte(r-'a'+'A'))
			case r >= 'A' && r <= 'A':
				letters = append(letters, byte(r))
			case unicode.IsSpace(r):
			default:
				return p, fmt.Errorf("input contains disallowed letter %q", r)
			}
		}
	}
	if len(letters) != gameSize {
		return p, fmt.Errorf("got %d letters; expected %d", len(letters), gameSize)
	}
	for i, l0 := range letters {
		for j := i + 1; j < len(letters); j++ {
			if l0 == letters[j] {
				return p, fmt.Errorf("input contains duplicate letter %q", l0)
			}
		}
	}
	p.center = makeLetterSet(string(letters[:1]))
	p.all = makeLetterSet(string(letters))
	return p, nil
}

func (p puzzle) matches(letters letterSet) bool {
	return p.center&letters == p.center && p.all&letters == letters
}

type wordAndScore struct {
	word  string
	score int // lower is better
}

type wordGroup struct {
	letters letterSet
	words   []wordAndScore
}

func loadCorpus(name string) ([]wordGroup, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	groupsBySet := make(map[letterSet]*wordGroup)
	var score int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.ToUpper(scanner.Text())
		letters := makeLetterSet(word)
		group, ok := groupsBySet[letters]
		if !ok {
			group = &wordGroup{letters: letters}
			groupsBySet[letters] = group
		}
		group.words = append(group.words, wordAndScore{word, score})
		score++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var corpus []wordGroup
	for _, group := range groupsBySet {
		corpus = append(corpus, *group)
	}
	sort.Slice(corpus, func(i, j int) bool {
		return corpus[i].letters < corpus[j].letters
	})
	return corpus, nil
}
