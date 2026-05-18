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
	"github.com/stripe/stripe-go/v85"
)

func main() {
	cfg := config.New()
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	defer db.Close()
	queries := database.New(db)

	r := NewRouter(cfg)

	// Services
	authService := services.NewAuthService(cfg.JWTSecret)
	stripeClient := stripe.NewClient(cfg.StripeSecretKey)
	stripeService := services.NewStripeWebhookService(queries)
	bookingService := services.NewBookingService(queries, stripeClient, cfg.StripeCurrency)

	// Handlers
	servicesHandler := handlers.NewServicesHandler(queries)
	authHandler := handlers.NewAuthHandler(queries, authService)
	availabilityHandler := handlers.NewServiceAvailabilityHandler(queries)
	stripeWebhookHandler := handlers.NewStripeWebhookHandler(cfg.StripeWebhookSecret, stripeService)
	bookingsHandler := handlers.BookingsHandler{BookingService: bookingService}

	r.Get("/healthz", handlers.HealthHandler)
	r.Route("/v1/api", func(r chi.Router) {
		// Webhooks
		r.Post("/webhooks/stripe", stripeWebhookHandler.HandleStripeWebHook)

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

				// Bookings
				r.Post("/bookings", bookingsHandler.CreateBooking)
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

				// Availability routes
				r.Get("/availability", availabilityHandler.GetAllAvailability)
				r.Get("/availability/{id}", availabilityHandler.GetAvailability)
				r.Get("/services/{service_id}/availability", availabilityHandler.GetAvailabilityByService)
				r.Post("/availability", availabilityHandler.CreateAvailability)
				r.Patch("/availability/{id}", availabilityHandler.UpdateAvailability)
				r.Delete("/availability/{id}", availabilityHandler.DeleteAvailability)
			})
	})

	srv := NewServer(r, cfg)
	srv.Start()
}
