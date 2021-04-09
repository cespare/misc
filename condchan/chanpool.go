package main

import (
	"context"
	"sync"
)

type token struct{}

type ChanPool struct {
	size int
	sem  chan token // buffered to size; send to acquire
	idle chan *Conn // buffered to size; send and receive with sem acquired

	mu     sync.Mutex
	closed chan struct{} // closed with mu held when the pool is closed
}

func NewChanPool(size int) *ChanPool {
	return &ChanPool{
		size:   size,
		sem:    make(chan token, size),
		idle:   make(chan *Conn, size),
		closed: make(chan struct{}),
	}
}

func (p *ChanPool) Get() (conn *Conn, err error) {
	ctx := context.TODO()
	select {
	case <-p.closed:
		return nil, ErrPoolClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.sem <- token{}:
	}
	defer func() {
		if conn == nil {
			<-p.sem
		}
	}()

	select {
	case conn = <-p.idle:
		return conn, nil
	default:
		return Dial()
	}
}

func (p *ChanPool) Put(c *Conn) {
	if !c.closed {
		p.idle <- c
	}
	<-p.sem // Release after c is closed or returned.
}

func (p *ChanPool) Close() {
	p.mu.Lock()
	select {
	case <-p.closed:
		p.mu.Unlock()
		return
	default:
		close(p.closed)
	}
	p.mu.Unlock()

	// Wait for all connections to be returned to the pool.
	for n := p.size; n > 0; n-- {
		p.sem <- token{}
	}
	close(p.idle)
	for c := range p.idle {
		c.Close()
	}
}
