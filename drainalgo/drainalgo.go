// drainalgo is a synthetic test for various pool-draining heuristics discussed
// in https://github.com/golang/go/issues/22950. This is intended to explore the
// effects of various strategies under relatively low get/put rates where
// overall hit rate is a problem, NOT high-contention scenarios.
package main

import (
	"flag"
	"log"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type pool interface {
	get() interface{}
	put(interface{})
	gc()
	size() int
}

type syncPool struct {
	p sync.Pool
	n int64
}

func (p *syncPool) get() interface{} {
	x := p.p.Get()
	if x != nil {
		atomic.AddInt64(&p.n, -1)
	}
	return x
}

func (p *syncPool) put(x interface{}) {
	p.p.Put(x)
	atomic.AddInt64(&p.n, 1)
}

func (p *syncPool) gc() {
	atomic.StoreInt64(&p.n, 0)
}

func (p *syncPool) size() int {
	return int(atomic.LoadInt64(&p.n))
}

type simpleSyncPool struct {
	mu    sync.Mutex
	items []interface{}
}

func (p *simpleSyncPool) get() interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.items)
	if n == 0 {
		return nil
	}
	x := p.items[n-1]
	p.items = p.items[:n-1]
	return x
}

func (p *simpleSyncPool) put(x interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, x)
}

func (p *simpleSyncPool) gc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = nil
}

func (p *simpleSyncPool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

type maxLivePool struct {
	mu      sync.Mutex
	items   []interface{}
	live    int
	maxLive int
}

func (p *maxLivePool) get() interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.live++
	if p.live > p.maxLive {
		p.maxLive = p.live
	}
	n := len(p.items)
	if n == 0 {
		return nil
	}
	x := p.items[n-1]
	p.items = p.items[:n-1]
	return x
}

func (p *maxLivePool) put(x interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.live--
	p.items = append(p.items, x)
}

func (p *maxLivePool) gc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) > p.maxLive {
		p.items = p.items[:p.maxLive]
	}
	p.maxLive = p.live
}

func (p *maxLivePool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

type minDeadPool struct {
	mu      sync.Mutex
	items   []interface{}
	minDead int
}

func (p *minDeadPool) get() interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.items)
	if n == 0 {
		return nil
	}
	x := p.items[n-1]
	p.items = p.items[:n-1]
	if len(p.items) < p.minDead {
		p.minDead = len(p.items)
	}
	return x
}

func (p *minDeadPool) put(x interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, x)
}

func (p *minDeadPool) gc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.minDead > len(p.items) {
		p.items = p.items[:0]
	} else {
		p.items = p.items[:len(p.items)-p.minDead]
	}
	p.minDead = len(p.items)
}

func (p *minDeadPool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

func main() {
	log.SetFlags(0)
	var (
		gcInterval   = flag.Duration("gcinterval", 4*time.Second, "How often to run a GC")
		poolType     = flag.String("pooltype", "sync.Pool", `What kind of pool to use ("sync.Pool", "simple", "maxlive", "mindead")`)
		getInterval  = flag.Duration("getinterval", 10*time.Millisecond, "Mean delay between gets")
		holdInterval = flag.Duration("holdinterval", 1*time.Millisecond, "Mean time each worker holds an item before returning it to the pool")
	)
	flag.Parse()

	var p pool
	switch *poolType {
	case "sync.Pool":
		// Use the real sync.Pool implementation.
		p = new(syncPool)
	case "simple":
		// This is a simplistic simulation of the sync.Pool
		// implementation. Items are added to a big shared pool and the
		// whole pool is cleared on each collection. Unlike the real
		// sync.Pool, there are no per-P private items.
		p = new(simpleSyncPool)
	case "maxlive":
		// This simulates the "max live set" heuristic I described in
		// https://github.com/golang/go/issues/22950#issuecomment-348653747:
		// at each GC, we trim the pool down to the maximum live set
		// size during the cycle.
		p = new(maxLivePool)
	case "mindead":
		// This simulates the "min dead set" heuristic described by Austin in
		// https://github.com/golang/go/issues/22950#issuecomment-352935997:
		// at each GC, we evict as many items from the pool as was the
		// minimum occupancy of the pool during the cycle.
		p = new(minDeadPool)
	default:
		log.Fatalf("Unknown pool type %q", *poolType)
	}

	// Spin up work to mostly saturate CPU to ensure that our
	// "event-handling" goroutines, below, actually get scheduled across the
	// available Ps.
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			for {
				start := time.Now()
				var n int64
				for i := 0; i < 1e6; i++ {
					n++
				}
				time.Sleep(time.Since(start) / 10)
			}
		}()
	}

	var numGCs int64
	go func() {
		debug.SetGCPercent(-1)
		for range time.Tick(*gcInterval) {
			runtime.GC()
			p.gc()
			atomic.AddInt64(&numGCs, 1)
		}
	}()

	var (
		sizeMu            sync.Mutex
		totalObservedSize float64
		sizeObservations  float64
	)
	// Sample the size every 100ms.
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			sizeMu.Lock()
			totalObservedSize += float64(p.size())
			sizeObservations += 1
			sizeMu.Unlock()
		}
	}()

	var hits, misses int64
	go func() {
		for {
			time.Sleep(getDelay(*getInterval))
			go func() {
				x := p.get()
				if x == nil {
					atomic.AddInt64(&misses, 1)
					x = new(int)
				} else {
					atomic.AddInt64(&hits, 1)
				}
				time.Sleep(holdDelay(*holdInterval))
				p.put(x)
			}()
		}
	}()
	t := time.Now()
	for range time.Tick(3 * time.Second) {
		hits := atomic.LoadInt64(&hits)
		misses := atomic.LoadInt64(&misses)
		gets := hits + misses
		numGCs := atomic.LoadInt64(&numGCs)
		secs := time.Since(t).Seconds()
		sizeMu.Lock()
		avgPoolSize := totalObservedSize / sizeObservations
		sizeMu.Unlock()
		log.Printf(
			"%d gets (%.1f/sec) / %.1f%% hit rate / %.1f avg live size / %d GCs (%.3f/sec)",
			gets, float64(gets)/secs,
			float64(hits)/float64(gets)*100,
			avgPoolSize,
			numGCs, float64(numGCs)/secs,
		)
	}
}

func getDelay(mean time.Duration) time.Duration {
	// Model the delay between gets as a Poisson process (exponentially
	// distributed delay).
	return time.Duration(rand.ExpFloat64() * float64(mean))
}

func holdDelay(mean time.Duration) time.Duration {
	// Use a normal distribution centered at the given mean with a standard
	// deviation of 10% of the mean.
	// (Though in most applications this distribution is probably quite skewed.)
	m := float64(mean)
	d := time.Duration(m + rand.NormFloat64()*0.1*m)
	if d < 0 {
		return 0
	}
	return d
}
