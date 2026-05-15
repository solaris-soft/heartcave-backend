package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/config"
)

func NewRouter(cfg *config.Config) chi.Router {
	r := chi.NewRouter()
	RegisterCommonMiddlewares(r, cfg)
	return r
}
