package http

import (
	"net/http"
)

type middle func(http.Handler) http.Handler

type Router interface {
	Handle(string, http.Handler)
	Group(string, func(r Router))
	Use(func(http.Handler) http.Handler)
}

type route struct {
	mux     *http.ServeMux
	pattern string
	middles []func(http.Handler) http.Handler
}

func NewRouter() *route {
	return &route{
		mux:     http.NewServeMux(),
		middles: make([]func(http.Handler) http.Handler, 0),
	}
}

func (r *route) Handle(pattern string, handler http.Handler) {
	for i := range r.middles {
		handler = r.middles[len(r.middles)-1-i](handler)
	}
	r.mux.Handle(r.pattern+pattern, handler)
}

func (r *route) Group(pattern string, fn func(r Router)) {
	if r.pattern != "" {
		pattern = pattern[1:]
	}
	r.pattern += pattern
	fn(r)
}

func (r *route) Use(middle func(http.Handler) http.Handler) {
	r.middles = append(r.middles, middle)
}
