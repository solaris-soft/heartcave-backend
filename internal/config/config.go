package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// getEnv returns the value from the environment or the fallback value if not set.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// mustEnv crashes the server if the environment variable is not present
func mustEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Fatal("Required key missing: ", key)
	return ""
}

// Config holds the configuration for the application
type Config struct {
	Addr                string
	Logger              *slog.Logger
	DBURL               string
	JWTSecret           string
	AppTimezone         string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeCurrency      string
}

// New reads the environment variables and returns the configuration
func New() *Config {
	godotenv.Load()
	address := getEnv("ADDRESS", ":8080")
	jwtSecret := mustEnv("SECRET_KEY")
	dbURL := mustEnv("DB_URL")

	return &Config{
		Addr:  address,
		DBURL: dbURL,
		Logger: slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		)),
		JWTSecret:           jwtSecret,
		AppTimezone:         getEnv("APP_TIMEZONE", "Australia/Perth"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeCurrency:      getEnv("STRIPE_CURRENCY", "aud"),
	}
}
