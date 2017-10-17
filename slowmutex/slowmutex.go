package main

import (
	"flag"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tinylib/spin"
)

func main() {
	n := flag.Int("n", 300, "number of workers")
	flag.Parse()

	s := newState()
	for i := 0; i < *n; i++ {
		go s.work()
	}

	for range time.Tick(3 * time.Second) {
		s.report()
	}
}

type state struct {
	//mu mutex
	mu sync.Mutex

	waiters    int64
	lastUnlock int64
	iters      int64
	delay      int64
	inside     int64
	unlock     int64
	sleep      int64
	maxDelay   int64
	minWaiters int64

	lastReport time.Time
}

func newState() *state {
	return &state{
		lastReport: time.Now(),
		minWaiters: -1,
	}
}

func (s *state) report() {
	t := time.Now()
	secs := t.Sub(s.lastReport).Seconds()
	s.lastReport = t

	iters := atomic.SwapInt64(&s.iters, 0)
	itersPerSec := float64(iters) / secs
	delay := atomic.SwapInt64(&s.delay, 0)
	meanDelay := time.Duration(delay / iters)
	inside := time.Duration(atomic.SwapInt64(&s.inside, 0))
	insidePerSec := time.Duration(float64(inside) / secs)
	unlock := time.Duration(atomic.SwapInt64(&s.unlock, 0))
	unlockPerSec := time.Duration(float64(unlock) / secs)
	sleep := time.Duration(atomic.SwapInt64(&s.sleep, 0))
	sleepPerIter := time.Duration(float64(sleep) / float64(iters))
	maxDelay := time.Duration(atomic.SwapInt64(&s.maxDelay, 0))
	minWaiters := atomic.SwapInt64(&s.minWaiters, -1)
	log.Printf("%.1f iters/sec; min #waiters %d; time inside %s/s; time unlocking %s/s; sleep %s/s; mean delay %s; max delay %s",
		itersPerSec, minWaiters, insidePerSec, unlockPerSec, sleepPerIter, meanDelay, maxDelay)
}

func (s *state) work() {
	for {
		busyWork(10)
		atomic.AddInt64(&s.waiters, 1)
		s.mu.Lock()
		waiters := atomic.AddInt64(&s.waiters, -1)
		lastUnlock := atomic.LoadInt64(&s.lastUnlock)
		t0 := time.Now().UnixNano()
		if lastUnlock > 0 {
			d := t0 - lastUnlock
			atomic.AddInt64(&s.delay, d)
			setMax(&s.maxDelay, d)
		}
		setMin(&s.minWaiters, waiters)
		//time.Sleep(3000 * time.Nanosecond)
		runtime.Gosched()
		t1 := time.Now().UnixNano()
		s.mu.Unlock()
		t2 := time.Now().UnixNano()
		setMax(&s.lastUnlock, t2)
		atomic.AddInt64(&s.unlock, t2-t1)
		atomic.AddInt64(&s.sleep, t1-t0)
		atomic.AddInt64(&s.inside, t2-t0)
		atomic.AddInt64(&s.iters, 1)
	}
}

func setMax(p *int64, v int64) {
	for {
		cur := atomic.LoadInt64(p)
		if v <= cur {
			return
		}
		if atomic.CompareAndSwapInt64(p, cur, v) {
			return
		}
	}
}

func setMin(p *int64, v int64) {
	for {
		cur := atomic.LoadInt64(p)
		if cur >= 0 && cur <= v {
			return
		}
		if atomic.CompareAndSwapInt64(p, cur, v) {
			return
		}
	}
}

func busyWork(n int) {
	for i := 0; i < n; i++ {
		busy()
	}
}

func busy() {
	var n int64
	for i := 0; i < 1e4; i++ {
		n++
	}
}

type mutex struct {
	lock uint32
}

func (m *mutex) Lock() {
	spin.Lock(&m.lock)
}

func (m *mutex) Unlock() {
	spin.Unlock(&m.lock)
}
