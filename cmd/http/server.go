package main

import (
	"net/http"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/config"
)

type Server struct {
	router http.Handler
	config *config.Config
	server *http.Server
}

func NewServer(router http.Handler, config *config.Config) *Server {
	svr := http.Server{
		Addr:           config.Addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return &Server{
		router: router,
		config: config,
		server: &svr,
	}
}

func (s *Server) Start() {
	s.config.Logger.Info("Starting server", "address", s.config.Addr)
	if err := s.server.ListenAndServe(); err != nil {
		s.config.Logger.Error("error starting or shutting down server", "err", err)
	}
}
