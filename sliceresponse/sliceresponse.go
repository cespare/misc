package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"math/rand"
	"net"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

func main() {
	var (
		server = flag.Bool("server", false, "server mode")
		addr   = flag.String("addr", "localhost:8989", "addr")
		min    = flag.Int64("min", 0, "min")
		max    = flag.Int64("max", 100, "min")
		n      = flag.Int64("n", 100e3, "n")
		format = flag.String("format", "uvarint", "format")
		verify = flag.Bool("verify", false, "do verification")
	)
	flag.Parse()

	if *server {
		ln, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatal(err)
		}
		for {
			c, err := ln.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go handle(c)
		}
		panic("unreached")
	}

	c, err := net.Dial("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	bw := bufio.NewWriter(c)
	br := bufio.NewReader(c)
	var (
		mu      sync.Mutex
		elapsed time.Duration
		count   int64
	)
	go func() {
		for range time.Tick(5 * time.Second) {
			mu.Lock()
			log.Printf("%s / request (%d requests)",
				elapsed/time.Duration(count), count)
			elapsed = 0
			count = 0
			mu.Unlock()
		}
	}()
	req := &request{
		min: *min,
		max: *max,
		n:   *n,
	}
	switch *format {
	case "uvarint":
		req.format = formatUvarint
	case "bigendian":
		req.format = formatBigEndian
	case "unsafe":
		req.format = formatUnsafe
	default:
		log.Fatalln("bad format:", *format)
	}
	var rsp response
	for {
		if err := req.encode(bw); err != nil {
			log.Fatal(err)
		}
		if err := bw.Flush(); err != nil {
			log.Fatal(err)
		}
		start := time.Now()
		if err := rsp.decode(br); err != nil {
			log.Fatal(err)
		}
		mu.Lock()
		elapsed += time.Since(start)
		count++
		mu.Unlock()
		if *verify {
			if len(rsp.s) != int(*n) {
				log.Fatalf("verification failure: got response with len=%d", len(rsp.s))
			}
			for i, v := range rsp.s {
				if int64(v) < *min || int64(v) >= *max {
					log.Fatalf("verification failure: got value %d at i=%d", v, i)
				}
			}
		}
	}
}

func handle(c net.Conn) {
	var (
		req0  request
		req1  request
		first = true
		rsp   response
		w     io.Writer = c
		br              = bufio.NewReader(c)
	)
	for {
		if err := req0.decode(br); err != nil {
			log.Print(err)
			return
		}
		if first {
			first = false
			req1 = req0
			rsp.format = req0.format
			if rsp.format != formatUnsafe {
				w = bufio.NewWriter(c)
			}
			rsp.s = make([]uint64, req0.n)
			for i := range rsp.s {
				d := int(req0.max - req0.min)
				rsp.s[i] = uint64(rand.Intn(d) + int(req0.min))
			}
		} else if req0 != req1 {
			log.Print("params changed on a single connection")
			return
		}
		if err := rsp.encode(w); err != nil {
			log.Print(err)
			return
		}
		if bw, ok := w.(*bufio.Writer); ok {
			if err := bw.Flush(); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

const (
	formatUvarint   = 0
	formatBigEndian = 1
	formatUnsafe    = 2
)

type request struct {
	min    int64
	max    int64
	n      int64
	format byte
}

func (req *request) encode(w io.Writer) error {
	if err := writeUvarint(w, uint64(req.min)); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(req.max)); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(req.n)); err != nil {
		return err
	}
	_, err := w.Write([]byte{req.format})
	return err
}

func (req *request) decode(r reader) error {
	x, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	req.min = int64(x)
	x, err = binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	req.max = int64(x)
	x, err = binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	req.n = int64(x)
	req.format, err = readByte(r)
	return err
}

type response struct {
	format byte
	s      []uint64
}

func (rsp *response) encode(w io.Writer) error {
	if _, err := w.Write([]byte{rsp.format}); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(len(rsp.s))); err != nil {
		return err
	}
	switch rsp.format {
	case formatUvarint:
		b := make([]byte, binary.MaxVarintLen64)
		for _, v := range rsp.s {
			n := binary.PutUvarint(b, v)
			if _, err := w.Write(b[:n]); err != nil {
				return err
			}
		}
	case formatBigEndian:
		b := make([]byte, 8)
		for _, v := range rsp.s {
			binary.BigEndian.PutUint64(b, v)
			if _, err := w.Write(b); err != nil {
				return err
			}
		}
	case formatUnsafe:
		var b []byte
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
		hdr.Len = 8 * len(rsp.s)
		hdr.Cap = hdr.Len
		hdr.Data = uintptr(unsafe.Pointer(&rsp.s[0]))
		if _, err := w.Write(b); err != nil {
			return err
		}
	default:
		panic("bad format")
	}
	return nil
}

func (rsp *response) decode(r reader) error {
	var err error
	rsp.format, err = readByte(r)
	if err != nil {
		return err
	}
	x, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	n := int(x)
	if cap(rsp.s) >= n {
		rsp.s = rsp.s[:n]
	} else {
		rsp.s = make([]uint64, n)
	}
	switch rsp.format {
	case formatUvarint:
		for i := range rsp.s {
			v, err := binary.ReadUvarint(r)
			if err != nil {
				return err
			}
			rsp.s[i] = v
		}
	case formatBigEndian:
		b := make([]byte, 8)
		for i := range rsp.s {
			if _, err := io.ReadFull(r, b); err != nil {
				return err
			}
			rsp.s[i] = binary.BigEndian.Uint64(b)
		}
	case formatUnsafe:
		var b []byte
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
		hdr.Len = 8 * len(rsp.s)
		hdr.Cap = hdr.Len
		hdr.Data = uintptr(unsafe.Pointer(&rsp.s[0]))
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
	default:
		panic("bad format")
	}
	return nil
}

type reader interface {
	io.Reader
	io.ByteReader
}

func readByte(r reader) (byte, error) {
	b := make([]byte, 1)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	return b[0], nil
}

func writeUvarint(w io.Writer, u uint64) error {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, u)
	_, err := w.Write(b[:n])
	return err
}
