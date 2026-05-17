package main

import (
	"database/sql"
	"log"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/solaris-soft/heartcave-backend/internal/config"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/handlers"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

func main() {
	cfg := config.New()
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
	queries := database.New(db)

	r := NewRouter(cfg)

	authService := services.NewAuthService(cfg.JWTSecret)

	authHandler := handlers.NewAuthHandler(queries, authService)

	r.Get("/healthz", handlers.HealthHandler)
	r.Route("/v1/api", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
		r.Post("/register", authHandler.CreateUser)
	})

	srv := NewServer(r, cfg)
	srv.Start()
}
