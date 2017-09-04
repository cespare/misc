package main

import (
	"sync"
)

type CondPool struct {
	size int

	mu     sync.Mutex // protects all following fields
	cond   *sync.Cond // linked to mu
	idle   []*Conn
	closed bool
	active int // while active < size, there are available slots
}

func NewCondPool(size int) *CondPool {
	p := &CondPool{size: size}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *CondPool) Get() (*Conn, error) {
	p.mu.Lock()
	for {
		if p.closed {
			p.mu.Unlock()
			// Let any other goroutine blocked in Get or in Close
			// know that they should re-check.
			p.cond.Broadcast()
			return nil, ErrPoolClosed
		}
		// First try to grab an idle connection.
		if n := len(p.idle); n > 0 {
			c := p.idle[n-1]
			p.idle = p.idle[:n-1]
			p.active++
			p.mu.Unlock()
			return c, nil
		}
		// If there is room to make a new one, do so.
		if p.active < p.size {
			// Unlock the mutex while dialing.
			p.active++
			p.mu.Unlock()
			c, err := Dial()
			if err != nil {
				p.mu.Lock()
				p.active--
				p.mu.Unlock()
				return nil, err
			}
			return c, nil
		}
		// We have to wait for a connection to become available.
		p.cond.Wait()
	}
}

// put returns a conn to the pool. If the conn is closed,
// it will be discarded instead.
func (p *CondPool) Put(c *Conn) {
	p.mu.Lock()
	if !c.closed {
		p.idle = append(p.idle, c)
	}
	p.active--
	p.mu.Unlock()
	p.cond.Signal()
}

func (p *CondPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	// Wait for all connections to be returned to the pool.
	for p.active > 0 {
		p.cond.Broadcast()
		p.cond.Wait()
	}
	for _, c := range p.idle {
		c.Close()
	}
	p.idle = nil
}
