package main

import (
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

	// Handlers
	blogHandler := handlers.NewBlogHandler(queries, cfg.Logger)
	customerHandler := handlers.NewCustomerHandler(queries, cfg.Logger)

	router := NewRouter()

	// Blog
	router.Post("/blog", blogHandler.CreateBlogPost)
	router.Get("/blog/{id}", blogHandler.GetBlogPostByID)
	router.Get("/blog", blogHandler.GetBlogPosts)

	// Customers
	router.Post("/customer", customerHandler.CreateCustomer)
	router.Get("/customer", customerHandler.GetCustomers)

	// Intialise server
	srv := NewServer(router, cfg)
	srv.Start()
}
