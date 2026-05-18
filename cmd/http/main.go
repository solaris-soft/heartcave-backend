package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	_ "github.com/lib/pq"
	"github.com/solaris-soft/heartcave-backend/internal/config"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/handlers"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

func main() {
	cfg := config.New()
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil || db.Ping() != nil {
		log.Fatal("Failed to connect to database")
	}
	defer db.Close()
	queries := database.New(db)

	r := NewRouter(cfg)

	authService := services.NewAuthService(cfg.JWTSecret)

	authHandler := handlers.NewAuthHandler(queries, authService)

	servicesHandler := handlers.NewServicesHandler(queries)

	r.Get("/healthz", handlers.HealthHandler)
	r.Route("/v1/api", func(r chi.Router) {
		// Rate limited routes
		r.With(httprate.LimitByIP(10, time.Minute)).
			Group(func(r chi.Router) {
				r.Post("/login", authHandler.Login)
				r.Post("/register", authHandler.CreateUser)
			})

		// Auth routes
		r.With(authHandler.AuthMiddleware).
			Group(func(r chi.Router) {
				r.Post("/logout", authHandler.Logout)
				r.Put("/users", authHandler.UpdateUser)
			})

		// Refresh token route
		r.With(authHandler.RefreshMiddleware).
			Post("/refresh", authHandler.Refresh)

		// Admin-only routes
		r.With(authHandler.AuthMiddleware, handlers.RequireRole("admin")).
			Group(func(r chi.Router) {
				// Services routes
				r.Get("/services", servicesHandler.GetServices)
				r.Get("/services/{id}", servicesHandler.GetService)
				r.Post("/services", servicesHandler.CreateService)
				r.Patch("/services/{id}", servicesHandler.UpdateService)
				r.Delete("/services/{id}", servicesHandler.DeleteService)
			})
	})

	srv := NewServer(r, cfg)
	srv.Start()
}
