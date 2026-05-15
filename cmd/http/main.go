package main

import (
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/config"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/handlers"
	_ "modernc.org/sqlite"
)

func main() {
	// Initiliase config
	cfg := config.New()

	// Initialise database
	database := cfg.Database.Prepare()
	defer database.Close()
	queries := db.New(database)

	location, err := time.LoadLocation(cfg.AppTimezone)
	if err != nil {
		cfg.Logger.Warn("unable to load app timezone; falling back to local", "timezone", cfg.AppTimezone, "err", err)
		location = time.Local
	}

	// Handlers
	authHandler := handlers.NewAuthHandler(queries, cfg.Logger, cfg.CustomerJWTSecret, cfg.AdminJWTSecret, cfg.AdminEmail, cfg.AdminPassword)
	blogHandler := handlers.NewBlogHandler(queries, cfg.Logger)
	servicesHandler := handlers.NewServicesHandler(queries, cfg.Logger)
	customersHandler := handlers.NewCustomersHandler(queries, cfg.Logger)
	scheduleHandler := handlers.NewScheduleHandler(queries, cfg.Logger)
	availabilityHandler := handlers.NewAvailabilityHandler(queries, cfg.Logger, location)
	bookingsHandler := handlers.NewBookingsHandler(
		database,
		queries,
		cfg.Logger,
		location,
		cfg.StripeSecretKey,
		cfg.StripeWebhookSecret,
		cfg.StripeCurrency,
		cfg.FrontendURL,
	)

	router := NewRouter(cfg)

	// Public API
	router.Get("/blog", blogHandler.ListPublished)
	router.Get("/blog/{slug}", blogHandler.GetPublishedBySlug)
	router.Get("/services", servicesHandler.List)
	router.Get("/availability", availabilityHandler.Get)
	router.Post("/auth/register", authHandler.Register)
	router.Post("/auth/login", authHandler.Login)
	router.Post("/admin/login", authHandler.AdminLogin)
	router.Post("/stripe/webhook", bookingsHandler.StripeWebhook)

	// Customer API
	router.With(handlers.CustomerRequired(cfg.CustomerJWTSecret)).Get("/me/bookings", bookingsHandler.ListMine)
	router.With(handlers.CustomerRequired(cfg.CustomerJWTSecret)).Post("/me/bookings", bookingsHandler.Create)

	// Admin API
	admin := router.With(handlers.AdminRequired(cfg.AdminJWTSecret))
	admin.Get("/admin/bookings", bookingsHandler.AdminList)
	admin.Get("/admin/customers", customersHandler.List)

	admin.Get("/admin/blog", blogHandler.AdminList)
	admin.Post("/admin/blog", blogHandler.AdminCreate)
	admin.Get("/admin/blog/{id}", blogHandler.AdminGet)
	admin.Put("/admin/blog/{id}", blogHandler.AdminUpdate)
	admin.Delete("/admin/blog/{id}", blogHandler.AdminDelete)

	admin.Get("/admin/schedule", scheduleHandler.List)
	admin.Post("/admin/schedule", scheduleHandler.Create)
	admin.Put("/admin/schedule/{id}", scheduleHandler.Update)
	admin.Delete("/admin/schedule/{id}", scheduleHandler.Delete)

	admin.Post("/admin/services", servicesHandler.Create)
	admin.Put("/admin/services/{id}", servicesHandler.Update)
	admin.Delete("/admin/services/{id}", servicesHandler.Delete)

	// Intialise server
	srv := NewServer(router, cfg)
	srv.Start()
}
