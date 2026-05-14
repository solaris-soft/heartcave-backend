package main

import (
	"log"

	"github.com/pressly/goose/v3"
	migrate "github.com/solaris-soft/heartcave-backend/db"
	"github.com/solaris-soft/heartcave-backend/internal/config"
	_ "modernc.org/sqlite"
)

func main() {
	cfg := config.New()
	database := cfg.Database.Prepare()
	defer database.Close()

	goose.SetBaseFS(migrate.Migrations)

	if err := goose.SetDialect("sqlite"); err != nil {
		log.Fatal(err)
	}

	if err := goose.Up(database, "migrations"); err != nil {
		log.Fatal(err)
	}

	log.Println("migrations applied successfully")
}
