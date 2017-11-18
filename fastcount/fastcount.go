package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type counter interface {
	inc()
	getAndZero() int64
}

type mutexCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *mutexCounter) inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

func (c *mutexCounter) getAndZero() int64 {
	c.mu.Lock()
	n := c.n
	c.n = 0
	c.mu.Unlock()
	return n
}

type atomicCounter struct {
	n int64
}

func (c *atomicCounter) inc()              { atomic.AddInt64(&c.n, 1) }
func (c *atomicCounter) getAndZero() int64 { return atomic.SwapInt64(&c.n, 0) }

const cacheLineSize = 64

type counterShard struct {
	n   int64
	pad [cacheLineSize - 8]byte
}

type shardedCounter struct {
	shards []counterShard
}

func newShardedCounter() shardedCounter {
	return shardedCounter{shards: make([]counterShard, numCores)}
}

func (c shardedCounter) inc() {
	i := getCore()
	atomic.AddInt64(&c.shards[i].n, 1)
}

func (c shardedCounter) getAndZero() int64 {
	var n int64
	for i := range c.shards {
		n += atomic.SwapInt64(&c.shards[i].n, 0)
	}
	return n
}

var numCores int

func init() {
	// Quick 'n' dirty way to count cores: scan /proc/cpuinfo.
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			numCores++
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if numCores == 0 {
		panic("no cores found (error processing /proc/cpuinfo)")
	}
}

func getCore() int

func main() {
	typ := flag.String("type", "mutex", "counter type")
	flag.Parse()

	var c counter
	switch *typ {
	case "mutex":
		c = new(mutexCounter)
	case "atomic":
		c = new(atomicCounter)
	case "sharded":
		c = newShardedCounter()
	default:
		log.Fatalf("unknown counter type %q", *typ)
	}

	numWorkers := runtime.GOMAXPROCS(0)
	fmt.Println("num workers:", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				for i := 0; i < 10000; i++ {
					c.inc()
				}
				runtime.Gosched()
			}
		}()
	}

	const interval = 5 * time.Second
	for range time.Tick(interval) {
		n := c.getAndZero()
		incsPerSec := float64(n) / interval.Seconds()
		latency := (interval / time.Duration(n)) * time.Duration(numWorkers)
		fmt.Printf("%.0f incs/sec; %.0f incs/sec/worker; avg latency %s\n",
			incsPerSec, incsPerSec/float64(numWorkers), latency)
	}
}
