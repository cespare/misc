package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	log.SetFlags(0)
	workaround := flag.String("workaround", "", `workaround to apply ("retry" or "flock")`)
	flag.Parse()

	switch *workaround {
	case "", "retry", "flock":
	default:
		log.Fatalf("Unknown -retry value %q", *workaround)
	}

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatalln("Error making tempdir:", err)
	}
	defer os.RemoveAll(dir)

	r := &runner{
		workaround: *workaround,
		dir:        dir,
	}
	r.run()
}

type runner struct {
	workaround string
	dir        string
}

func (r *runner) run() {
	var (
		trials   atomic.Int64
		failures atomic.Int64
		wg       sync.WaitGroup
	)
	for workerID := range 20 {
		wg.Go(func() {
			for range 1000 {
				trials.Add(1)
				if err := r.runOne(workerID); err != nil {
					if failures.Add(1) == 1 {
						log.Println("First failure:", err)
					}
				}
			}
		})
	}
	wg.Wait()
	log.Printf("%d/%d failed", failures.Load(), trials.Load())
}

func (r *runner) runOne(workerID int) error {
	text := []byte("#!/bin/bash\necho 'hello!'\n")
	name := filepath.Join(
		r.dir,
		fmt.Sprintf("%d.sh", workerID),
	)
	if r.workaround == "flock" {
		if err := writeFileSafe(name, text, 0o755); err != nil {
			return err
		}
	} else {
		if err := os.WriteFile(name, text, 0o755); err != nil {
			return err
		}
	}
	c := exec.Command(name)
	if r.workaround == "retry" {
		if err := startWithBusyRetries(c); err != nil {
			return err
		}
	} else {
		if err := c.Start(); err != nil {
			return err
		}
	}
	if err := c.Wait(); err != nil {
		return err
	}
	return nil
}

func startWithBusyRetries(c *exec.Cmd) error {
	var err error
	delay := 10 * time.Millisecond
	for i := range 10 {
		if i > 0 {
			time.Sleep(delay)
			delay = min(delay*2, time.Second)
		}
		err = c.Start()
		if !errors.Is(err, unix.ETXTBSY) {
			return err
		}
	}
	return err
}

func writeFileSafe(name string, text []byte, perm os.FileMode) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(text); err != nil {
		f.Close()
		return err
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	f, err = os.OpenFile(name, os.O_RDONLY, perm)
	if err != nil {
		return err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_SH); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
