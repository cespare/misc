package rwmutex

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type Client struct {
	n int
}

func (c *Client) MakeRequest() {
	time.Sleep(time.Nanosecond)
}

type S struct {
	mu          sync.Mutex
	mutexClient *Client

	rwmu          sync.RWMutex
	rwMutexClient *Client

	atomicClient atomic.Value // *Client
}

func New() *S {
	s := &S{
		mutexClient:   new(Client),
		rwMutexClient: new(Client),
	}
	s.atomicClient.Store(new(Client))
	return s
}

func (s *S) OpMutex() {
	s.mu.Lock()
	c := s.mutexClient
	s.mu.Unlock()
	c.MakeRequest()
}

func (s *S) OpRWMutex() {
	s.rwmu.RLock()
	c := s.rwMutexClient
	s.rwmu.RUnlock()
	c.MakeRequest()
}

func (s *S) OpAtomic() {
	c := s.atomicClient.Load().(*Client)
	c.MakeRequest()
}

func BenchmarkMutex(b *testing.B) {
	s := New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.OpMutex()
		}
	})
}

func BenchmarkRWMutex(b *testing.B) {
	s := New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.OpRWMutex()
		}
	})
}

func BenchmarkAtomic(b *testing.B) {
	s := New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.OpAtomic()
		}
	})
}
