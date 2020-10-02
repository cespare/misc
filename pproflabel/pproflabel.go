package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime/pprof"
	"time"

	"github.com/felixge/fgprof"
)

func main() {
	ctx := context.Background()
	go pprof.Do(ctx, pprof.Labels("mylabel", "A"), A)
	go pprof.Do(ctx, pprof.Labels("mylabel", "B"), B)
	http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())
	log.Fatal(http.ListenAndServe("localhost:8787", nil))
}

func A(_ context.Context) {
	x := 1234.567
	for {
		y := x / 5.678
		x += y
		time.Sleep(time.Microsecond)
	}
}

func B(_ context.Context) {
	x := 1234.567
	for {
		y := x / 5.678
		x += y
		time.Sleep(time.Microsecond)
	}
}
