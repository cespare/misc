package main

import (
	"sync"
)

type ChanPool struct {
	size int

	// ch has a buffer size of 1, and unless the pool is closed, ch only
	// contains a value if there are available slots in the pool
	// (i.e., if there is a state in ch with state.closed == false,
	// then state.avail > 0).
	ch chan chanPoolState

	// The idle stack is protected by mu, which should only be locked when
	// pushing/popping the slice.
	mu   sync.Mutex
	idle []*Conn

	// closeCh is closed when the pool is closed so that goroutines waiting
	// in Get for an available slot can wake up and return errPoolClosed.
	// This channel only exists to make closing fast -- it is not otherwise
	// needed for correctness.
	closeCh chan struct{}
}

type chanPoolState struct {
	avail  int
	closed bool
}

func NewChanPool(size int) *ChanPool {
	p := &ChanPool{
		size:    size,
		ch:      make(chan chanPoolState, 1),
		closeCh: make(chan struct{}),
	}
	p.ch <- chanPoolState{avail: size}
	return p
}

func (p *ChanPool) Get() (*Conn, error) {
	var state chanPoolState
	select {
	// In order to claim an available slot in the pool, a Get request must
	// grab the state from the p.ch. If the state isn't closed,
	// there are >0 available slots, and we'll return avail-1 of them
	// back to the pool below.
	case state = <-p.ch:
		if state.closed {
			return nil, ErrPoolClosed
		}
	case <-p.closeCh:
		return nil, ErrPoolClosed
	}
	state.avail--
	p.mu.Lock()
	if n := len(p.idle); n > 0 {
		c := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		p.mergeState(state)
		return c, nil
	}
	p.mu.Unlock()
	p.mergeState(state)
	c, err := Dial()
	if err != nil {
		// Return our single slot to the pool.
		p.mergeState(chanPoolState{avail: 1})
		return nil, err
	}
	return c, nil
}

// mergeState updates the state in p.ch. Any available slots in state are added
// to the existing slots.
func (p *ChanPool) mergeState(state chanPoolState) {
	for state.avail > 0 || state.closed {
		select {
		case p.ch <- state:
			return
		case state1 := <-p.ch:
			state.avail += state1.avail
			state.closed = state.closed || state1.closed
		}
	}
}

func (p *ChanPool) Put(c *Conn) {
	if !c.closed {
		p.mu.Lock()
		p.idle = append(p.idle, c)
		p.mu.Unlock()
	}
	p.mergeState(chanPoolState{avail: 1})
}

func (p *ChanPool) Close() {
	// Close closeCh so that any goroutines waiting in Get return right away.
	close(p.closeCh)
	// Introduce the closed state into p.ch.
	//
	// It's possible that a future state in p.ch won't have closed == true
	// (because a Get could grab the state and then a Put could insert the
	// state {1, false}). But in that case, the state with closed == true
	// still exists -- it's held by a goroutine in Get -- and it will
	// eventually be returned to p.ch.
	//
	// Even without closeCh, we'll eventually converge to zero open
	// connections because Puts will continue apace while Gets will see that
	// the pool is closed (with some probability).
	p.mergeState(chanPoolState{closed: true})
	// Wait until all open connections are closed by consuming all the slots.
	for remain := p.size; remain > 0; remain -= (<-p.ch).avail {
	}
	// Nobody else can access idle at this point; no need for lock.
	for _, c := range p.idle {
		c.Close()
	}
}
