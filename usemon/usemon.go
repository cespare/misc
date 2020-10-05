package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) < 2 {
		if err := usemon(); err != nil {
			log.Fatal(err)
		}
		return
	}
	switch os.Args[1] {
	case "parent0":
		if err := parent0(); err != nil {
			log.Fatal(err)
		}
	case "parent1":
		if err := parent1(); err != nil {
			log.Fatal(err)
		}
	case "mem":
		if err := useMemory(); err != nil {
			log.Fatal(err)
		}
	case "cpu":
		if err := useCPU(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("bad arg")
	}
}

type processStats struct {
	elapsed     time.Duration
	cpuUsage    time.Duration // utime+stime
	maxRSSBytes int64
}

func (ps *processStats) String() string {
	return fmt.Sprintf(
		"elapsed: %s, cpu: %s, max RSS: %s",
		ps.elapsed.Round(100*time.Millisecond),
		ps.cpuUsage.Round(100*time.Millisecond),
		humanize.Bytes(uint64(ps.maxRSSBytes)),
	)
}

func processTreeRSS(pgid int) (int64, error) {
	f, err := os.Open("/proc")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	dirs, err := f.Readdirnames(-1)
	if err != nil {
		return 0, err
	}
	var total int64
	var b []byte
	for _, d := range dirs {
		if !isDigits(d) {
			continue
		}
		var rss int64
		var err error
		rss, b, err = readRSS(d, pgid, b)
		if err != nil {
			switch err {
			case errWrongProcessGroup:
			default:
				log.Fatal(err)
			}
			// Many normal errors here -- we don't have permission to look
			// at root processes, the process has gone away, the
			// process is not in the process group we care about, etc.
			continue
		}
		total += rss
	}
	return total, nil
}

var (
	errWrongProcessGroup = errors.New("process not in target process group")
	errProcStatMalformed = errors.New("/proc/[pid]/stat line seems malformed")
)

func readRSS(pidStr string, pgid int, b []byte) (int64, []byte, error) {
	f, err := os.Open("/proc/" + pidStr + "/stat")
	if err != nil {
		return 0, b, err
	}
	defer f.Close()

	b, err = readAll(f, b)
	if err != nil {
		return 0, b, err
	}

	rss, err := parseStatRSS(pgid, b)
	if err != nil {
		return 0, b, err
	}
	return rss, b, nil
}

func parseStatRSS(pgid int, b []byte) (int64, error) {
	i := bytes.LastIndexByte(b, ')')
	if i < 0 {
		return 0, errProcStatMalformed
	}
	b = b[i+1:]
	if len(b) > 0 && b[0] == ' ' {
		b = b[1:]
	}
	var start, n int
	for i, c := range b {
		if c != ' ' {
			continue
		}
		switch n {
		case 2:
			id, err := strconv.Atoi(string(b[start:i]))
			if err != nil {
				return 0, err
			}
			if id != pgid {
				return 0, errWrongProcessGroup
			}
		case 21:
			rss, err := strconv.ParseInt(string(b[start:i]), 10, 64)
			if err != nil {
				return 0, err
			}
			return rss * 4096, nil
		}
		start = i + 1
		n++
	}
	return 0, errProcStatMalformed
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// readAll is a helper for reading the contents of a file into a reused slice.
// It attempts to use a single ReadAt to get the entire contents with a single
// syscall and falls back to ioutil.ReadAll in other cases.
func readAll(f *os.File, b []byte) ([]byte, error) {
	b = b[:cap(b)]
	if len(b) > 0 {
		// Attempt to do a single ReadAt for the common case.
		n, err := f.ReadAt(b, 0)
		if err == nil || err != io.EOF {
			return b[:n], err
		}
	}
	// Not enough buffer for ReadAt; fall back to ReadAll.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func run(cmd string, args ...string) (*processStats, error) {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	start := time.Now()
	if err := c.Start(); err != nil {
		return nil, err
	}
	var maxRSS int64
	rssMonCh := make(chan struct{}, 1)
	go func() {
		defer close(rssMonCh)
		const delay = 500 * time.Millisecond
		t := time.NewTimer(delay)
		defer t.Stop()
		for {
			select {
			case <-t.C:
			case <-rssMonCh:
				return
			}
			rss, err := processTreeRSS(c.Process.Pid)
			if err != nil {
				log.Println("Error finding RSS of process tree:", err)
				return
			}
			if rss > maxRSS {
				maxRSS = rss

			}
			t.Reset(delay)
		}
	}()
	err := c.Wait()
	rssMonCh <- struct{}{}
	<-rssMonCh
	rusage := c.ProcessState.SysUsage().(*syscall.Rusage)
	if rss := rusage.Maxrss * 1024; rss > maxRSS {
		maxRSS = rss
	}
	stats := &processStats{
		elapsed:     time.Since(start),
		cpuUsage:    time.Duration(rusage.Stime.Nano() + rusage.Utime.Nano()),
		maxRSSBytes: maxRSS,
	}
	return stats, err
}

func usemon() error {
	stats, err := run("./usemon", "parent0")
	if stats != nil {
		fmt.Println(stats)
	}
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func parent0() error {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "parent1").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "mem").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "cpu").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
	return nil
}

func parent1() error {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "mem").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "mem").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := exec.Command("./usemon", "cpu").Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
	return nil
}

func useMemory() error {
	s := make([]byte, 1e9)
	for i := range s {
		s[i] = byte(i)
	}
	time.Sleep(3 * time.Second)
	for i := 0; i < len(s); i += 1000 {
		fmt.Fprintln(ioutil.Discard, s[i])
	}
	return nil
}

func useCPU() error {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			end := time.Now().Add(3 * time.Second)
			for {
				var n int
				for i := 0; i < 1e6; i++ {
					n += i
				}
				fmt.Fprintln(ioutil.Discard, n)
				if time.Now().After(end) {
					return
				}
			}
		}()
	}
	wg.Wait()
	return nil
}
