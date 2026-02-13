package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	log.SetFlags(0)
	baseDir := os.TempDir()
	if len(os.Args) > 1 {
		baseDir = os.Args[1]
	}
	log.Println("Using base dir", baseDir)

	tmp, err := os.MkdirTemp(baseDir, "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	if err := touch(filepath.Join(tmp, "y")); err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	if err := unix.Rmdir(tmp); err == nil {
		log.Fatal("rmdir succeeded unexpectedly")
	}
	log.Println("First rmdir took", time.Since(start))

	for i := range 50_000 {
		name := filepath.Join(tmp, strconv.Itoa(i))
		if err := touch(name); err != nil {
			log.Fatal(err)
		}
		if err := os.Remove(name); err != nil {
			log.Fatal(err)
		}
	}

	start = time.Now()
	if err := unix.Rmdir(tmp); err == nil {
		log.Fatal("rmdir succeeded unexpectedly")
	}
	log.Println("Second rmdir took", time.Since(start))
}

func touch(name string) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
