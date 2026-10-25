package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Order service.
//
// Environment variables supplied by Docker, CI, or production take precedence
// over values loaded from the local .env file.
type Config struct {
	HTTPPort               string
	GRPCAddr               string
	StoreServiceAuthName   string
	StoreServiceAuthSecret string
	OrderServiceAuthSecret string
	OrderServiceAuthName   string
	CartServiceAddr        string
	AppEnv                 string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
}

// Load loads Order configuration.
//
// The .env file is a local-development convenience. Failure to load the file
// is not fatal because deployment environments can provide environment
// variables directly.
func Load() (*Config, error) {
	_ = godotenv.Load("services/order/.env")

	cfg := &Config{
		HTTPPort:               getEnv("ORDER_HTTP_PORT", "8094"),
		GRPCAddr:               getEnv("ORDER_GRPC_ADDR", ":9093"),
		StoreServiceAuthName:   getEnv("STORE_SERVICE_AUTH_NAME", "store"),
		StoreServiceAuthSecret: os.Getenv("STORE_SERVICE_AUTH_SECRET"),
		OrderServiceAuthSecret: os.Getenv("ORDER_SERVICE_AUTH_SECRET"),
		OrderServiceAuthName:   getEnv("ORDER_SERVICE_AUTH_NAME", "order"),
		CartServiceAddr:        getEnv("CART_SERVICE_ADDR", "localhost:9092"),
		AppEnv:                 getEnv("APP_ENV", "development"),
		DatabaseHost:           getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:           getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:           os.Getenv("DATABASE_USER"),
		DatabasePassword:       os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:           os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:        getEnv("DATABASE_SSL_MODE", "disable"),
	}

	if cfg.DatabaseUser == "" {
		return nil, fmt.Errorf("DATABASE_USER is required")
	}

	if cfg.DatabasePassword == "" {
		return nil, fmt.Errorf("DATABASE_PASSWORD is required")
	}

	if cfg.DatabaseName == "" {
		return nil, fmt.Errorf("DATABASE_NAME is required")
	}

	return cfg, nil
}

// DatabaseURL builds the PostgreSQL connection URL.
//
// url.URL and url.UserPassword ensure credentials and database names are
// escaped correctly instead of being concatenated into a raw URL.
func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			c.DatabaseUser,
			c.DatabasePassword,
		),
		Host: fmt.Sprintf("%s:%s", c.DatabaseHost, c.DatabasePort),
		Path: c.DatabaseName,
	}

	query := u.Query()
	query.Set("sslmode", c.DatabaseSSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}

// getEnv returns a nonempty environment value or its fallback.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
