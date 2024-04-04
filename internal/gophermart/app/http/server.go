package http

import (
	"context"
	"net/http"
)

type server struct {
	server *http.Server
}

type ServerConfig struct {
	Addr string
}

func NewServer(cfg ServerConfig, route route) server {
	return server{
		server: &http.Server{
			Addr:    cfg.Addr,
			Handler: route.mux,
		},
	}
}

func (s server) Start() error                   { return s.server.ListenAndServe() }
func (s server) Stop(ctx context.Context) error { return s.server.Shutdown(ctx) }
func (s server) RegisterShutdown(fn func())     { s.server.RegisterOnShutdown(fn) }
