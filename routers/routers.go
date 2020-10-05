package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v4"
	"github.com/gorilla/mux"
	"github.com/julienschmidt/httprouter"
)

func main() {
	eval("/x/y", "/x/y")
	eval("/x/y", "/x/y/")
	eval("/x/y/", "/x/y")
	eval("/x/y/", "/x/y/")
	eval("/x/y", "/x/z/../y")
	eval("/x/y", "/x//y")
	eval("/a", "/x//y")
	eval("/a", "/x/y/")
	evalWildcard("/a/", "/a/x/y")
	evalWildcard("/a/", "/a/")
	evalWildcard("/a/", "/a")

	evalParam("/x/{x}/y", "/x/a%2f%62c/y")
}

func eval(pat, path string) {
	h := msg("hello")
	chiRouter := chi.NewRouter()
	chiRouter.Put(pat, h)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(pat, h)

	hr := httprouter.New()
	hr.Handler("PUT", pat, h)

	gorilla := mux.NewRouter()
	gorilla.Handle(pat, h)

	fmt.Printf(
		"Pattern %s matches path %s:\n  chi: %s\n  ServeMux: %s\n  httprouter: %s\n  gorilla: %s\n",
		pat, path,
		matches(chiRouter, path),
		matches(serveMux, path),
		matches(hr, path),
		matches(gorilla, path),
	)
}

func evalWildcard(pat, path string) {
	h := msg("hello")
	chiRouter := chi.NewRouter()
	chiRouter.Put(pat+"*", h)

	hr := httprouter.New()
	hr.Handler("PUT", pat+"*xyz", h)

	fmt.Printf(
		"Pattern %s matches path %s:\n  chi: %s\n  httprouter: %s\n",
		pat+"*", path,
		matches(chiRouter, path),
		matches(hr, path),
	)
}

func evalParam(pat, path string) {
	chiRouter := chi.NewRouter()
	var chiParam string
	chiRouter.Put(pat, func(w http.ResponseWriter, r *http.Request) {
		chiParam = chi.URLParam(r, "x")
		w.Write([]byte("hello"))
	})
	chiMatches := matches(chiRouter, path)
	if chiParam != "" {
		chiMatches += fmt.Sprintf(" param=%q", chiParam)
	}

	hr := httprouter.New()
	hrPat := strings.ReplaceAll(pat, "{x}", ":x")
	h := msg("hello")
	hr.Handler("GET", hrPat, h)

	gorilla := mux.NewRouter()
	gorilla.Handle(pat, h)

	fmt.Printf(
		"Pattern %s matches path %s:\n  chi: %s\n  httprouter: %s\n  gorilla: %s\n",
		pat, path,
		chiMatches,
		matches(hr, path),
		matches(gorilla, path),
	)
}

func matches(router http.Handler, path string) string {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", path, nil)
	router.ServeHTTP(w, r)
	switch w.Code {
	case 404:
		return "no"
	case 301:
		loc := w.Result().Header.Get("Location")
		return fmt.Sprintf("301->%s", loc)
	case 307:
		loc := w.Result().Header.Get("Location")
		return fmt.Sprintf("307->%s", loc)
	case 405:
		return fmt.Sprintf("405 (%s)", w.Result().Header.Get("Allow"))
	case 200:
		if body := w.Body.String(); body != "hello" {
			panic(body)
		}
		return "yes"
	default:
		panic(w.Code)
	}
}

func matchesParam(router http.Handler, path string) string {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", path, nil)
	router.ServeHTTP(w, r)
	switch w.Code {
	case 404:
		return "no"
	case 301:
		loc := w.Result().Header.Get("Location")
		return fmt.Sprintf("301->%s", loc)
	case 200:
		if body := w.Body.String(); body != "hello" {
			panic(body)
		}
		return "yes"
	default:
		panic(w.Code)
	}
}

func msg(s string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(s))
	}
}
