package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Conn struct {
	// The real thing has a net.Conn and other stuff.
	id     int64
	closed bool
}

var connID int64

func Dial() (*Conn, error) {
	id := atomic.AddInt64(&connID, 1) - 1
	log.Printf("dial: created conn %d", id)
	return &Conn{id: id}, nil
}

func (c *Conn) Close() {}

// A Pool is the interface presented by a connection pool. In addition to the
// obvious behavior, it should satisfy the following properties:
//
// - A new connection should not be opened if there is an idle connection.
//   That is, 3 goroutines making use of a pool of size 10 should open at most 3
//   concurrent connections, not 10.
// - A closed connection may be returned to the pool; this should free up a slot
//   but the closed connection should not be cached.
// - Any Get request after the pool is closed should return ErrPoolClosed;
//   Puts should continue to work.
// - Close should interrupt any goroutine waiting in Get
//   (and make it return ErrPoolClosed).
// - Close should wait until all active connections are closed and then close
//   any cached (idle) connections.
type Pool interface {
	Get() (*Conn, error)
	Put(*Conn)
	Close()
}

var ErrPoolClosed = errors.New("pool is closed")

func main() {
	rand.Seed(time.Now().UnixNano())
	const (
		poolSize   = 3
		numWorkers = 5
	)
	//p := NewCondPool(poolSize)
	p := NewChanPool(poolSize)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go work(i, p, &wg)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("closing")
	p.Close()
	wg.Wait()
}

func work(i int, p Pool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		c, err := p.Get()
		if err == ErrPoolClosed {
			return
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("worker %d got conn %d", i, c.id)
		// Delay to simulate making a request on the connection.
		time.Sleep(time.Duration(rand.Intn(500)+500) * time.Millisecond)
		// Occasionally the connection might be broken.
		if rand.Intn(10) == 0 {
			c.closed = true
		}
		p.Put(c)
	}
}
