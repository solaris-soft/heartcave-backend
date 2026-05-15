package config

import (
	"database/sql"
	"log"
	"log/slog"
	"os"
)

// getEnv returns the value from the environment or the fallback value if not set.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Config holds the configuration for the application
type Config struct {
	Addr                string
	Logger              *slog.Logger
	Database            *SqliteConf
	AdminEmail          string
	AdminPassword       string
	CustomerJWTSecret   string
	AdminJWTSecret      string
	FrontendURL         string
	AllowedOrigin       string
	AppTimezone         string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeCurrency      string
}

// Config returns the logging configuration of the application based on the environment
func logOptions(env string) *slog.HandlerOptions {
	switch env {
	case "dev":
		return &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}
	default:
		return &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}
	}
}

// New reads the environment variables and returns the configuration
func New() *Config {
	env := getEnv("ENV", "dev")
	dbPath := getEnv("DB_PATH", "heartcave.db")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, logOptions(env)))
	address := getEnv("ADDRESS", ":8080")

	return &Config{
		Addr:   address,
		Logger: logger,
		Database: &SqliteConf{
			path: dbPath,
		},
		AdminEmail:          getEnv("ADMIN_EMAIL", "admin@heartcave.com"),
		AdminPassword:       getEnv("ADMIN_PASSWORD", "changeme"),
		CustomerJWTSecret:   getEnv("CUSTOMER_JWT_SECRET", getEnv("JWT_SECRET", "changeme-jwt-secret")),
		AdminJWTSecret:      getEnv("ADMIN_JWT_SECRET", getEnv("JWT_SECRET", "changeme-jwt-secret")),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:4321"),
		AllowedOrigin:       getEnv("ALLOWED_ORIGIN", getEnv("FRONTEND_URL", "http://localhost:4321")),
		AppTimezone:         getEnv("APP_TIMEZONE", "Australia/Perth"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeCurrency:      getEnv("STRIPE_CURRENCY", "aud"),
	}
}

// SqliteConf holds the configuration for the database
type SqliteConf struct {
	path string
}

// openDB opens the database
func (c *SqliteConf) openDB() *sql.DB {
	db, err := sql.Open("sqlite", c.path)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// setSettings sets the settings for the database
func (c *SqliteConf) setSettings(db *sql.DB) {
	// Connection pool tuning
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA cache_size = -20000;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA mmap_size = 268435456;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Fatal(err)
		}
	}
}

// Prepare the database for use
func (c *SqliteConf) Prepare() *sql.DB {
	db := c.openDB()
	c.setSettings(db)
	return db
}
