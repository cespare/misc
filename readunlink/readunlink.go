package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	n := flag.Int("n", 1000, "Number of iterations")
	flag.Parse()

	for i := 0; i < *n; i++ {
		if err := run(); err != nil {
			log.Fatalf("Failure after %d successful iterations: %s", i, err)
		}
	}
}

const filename = "x.txt"

func run() error {
	// Write a file.
	const msg = "hello!"
	if err := ioutil.WriteFile(filename, []byte(msg), 0o644); err != nil {
		return fmt.Errorf("error creating file for test: %s", err)
	}

	// Concurrently try to stat the file and unlink it.
	//
	// I would expect that either (a) the file doesn't exist when we stat it
	// or (b) the file exists and is 6 bytes.
	//
	// I would not expect to find an empty file, but sometimes that is the
	// result (so far only on macOS Catalina machines). Does that OS
	// truncate files as part of unlink or something?
	//
	// (The same result happens with open+read & unlink but the stat is
	// simpler for repro purposes.)

	done := make(chan struct{})
	go func() {
		defer close(done)

		if err := os.Remove(filename); err != nil {
			panic(err)
		}
	}()

	stat, err := os.Stat(filename)
	<-done

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// This is one of the expected outcomes.
			return nil
		}
		return fmt.Errorf("unexpected error stat-ing test file: %s", err)
	}
	if got, want := stat.Size(), int64(len(msg)); got != want {
		return fmt.Errorf("stat(%q) gave size %d; expected %d", filename, got, want)
	}

	// This is the other expected outcome.
	return nil

}
