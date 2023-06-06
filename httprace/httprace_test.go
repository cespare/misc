package httprace

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Why does the race detector say that the indicated lines have a data race?
func TestHTTPRace(t *testing.T) {
	var newConns int
	handler := func(w http.ResponseWriter, r *http.Request) {}
	server := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns++ // <-------------------------
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	client0 := &http.Client{Transport: &http.Transport{}}
	client1 := &http.Client{Transport: &http.Transport{}}

	get := func(client *http.Client) {
		t.Helper()
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	get(client0)
	get(client0)
	if newConns != 1 { // <-------------------------
		t.Fatalf("got newConns=%d; want 1", newConns)
	}
	get(client1)
}

// The race detector is happy with this one.
func TestHTTPRace2(t *testing.T) {
	var x int
	handler := func(w http.ResponseWriter, r *http.Request) {
		x++
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: &http.Transport{}}

	get := func() {
		t.Helper()
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	get()
	if x != 1 {
		t.Fatalf("got x=%d; want 1", x)
	}
	get()
}
