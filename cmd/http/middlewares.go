package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/solaris-soft/heartcave-backend/internal/config"
)

// RegisterCommonMiddlewares registers the middlewares that apply to the entire application
func RegisterCommonMiddlewares(r chi.Router, cfg *config.Config) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
}
