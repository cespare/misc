package main

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type ChanPool struct {
	size int
	sem  *semaphore.Weighted

	mu     sync.Mutex
	idle   chan *Conn // buffered to size; closed with mu held when the pool is closed
	closed bool
}

func NewChanPool(size int) *ChanPool {
	return &ChanPool{
		size: size,
		sem:  semaphore.NewWeighted(int64(size)),
		idle: make(chan *Conn, size),
	}
}

func (p *ChanPool) Get() (conn *Conn, err error) {
	if err := p.sem.Acquire(context.TODO(), 1); err != nil {
		return nil, err
	}
	defer func() {
		if conn == nil {
			p.sem.Release(1)
		}
	}()

	select {
	case conn, ok := <-p.idle:
		if !ok {
			return nil, ErrPoolClosed
		}
		return conn, nil
	default:
		return Dial()
	}
}

func (p *ChanPool) Put(c *Conn) {
	defer p.sem.Release(1) // Wait to release until c is closed or returned.
	if c.closed {
		return
	}

	p.mu.Lock()
	if p.closed {
		defer c.Close()
	} else {
		p.idle <- c
	}
	p.mu.Unlock()
}

func (p *ChanPool) Close() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.idle)
	}
	p.mu.Unlock()

	for conn := range p.idle {
		conn.Close()
	}
	// Wait for all connections to be returned to the pool.
	if err := p.sem.Acquire(context.TODO(), int64(p.size)); err == nil {
		p.sem.Release(int64(p.size))
	}
}
