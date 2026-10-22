package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Cart service.
type Config struct {
	HTTPAddr           string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	ProductServiceAddr string
	ServiceAuthName    string
	ServiceAuthSecret  string
}

// Load loads Cart configuration from environment variables.
// The .env file is a local-development convenience.
func Load() (*Config, error) {
	_ = godotenv.Load("services/cart/.env")

	cfg := &Config{
		HTTPAddr:           getEnv("HTTP_ADDR", ":8090"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		ProductServiceAddr: getEnv("PRODUCT_SERVICE_ADDR", "localhost:9091"),
		ServiceAuthName:    getEnv("SERVICE_AUTH_NAME", "cart"),
		ServiceAuthSecret:  os.Getenv("SERVICE_AUTH_SECRET"),
	}

	db, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil || db < 0 {
		return nil, fmt.Errorf("REDIS_DB must be a non-negative integer")
	}
	cfg.RedisDB = db

	if cfg.RedisPassword == "" {
		return nil, fmt.Errorf("REDIS_PASSWORD is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
