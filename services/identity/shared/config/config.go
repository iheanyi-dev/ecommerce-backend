package config

import (
	"fmt"
	"os"
	"net/url"
	"time"
	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
	// JWT configuration.
	JWTSecret           string
	JWTIssuer           string
	JWTAccessTokenTTL   time.Duration
}

func Load() (*Config, error) {
	// Load .env for local development.
	// In Docker/production, environment variables can be supplied
	// directly by the runtime.
	_ = godotenv.Load("services/identity/.env")
	jwtAccessTokenTTL, err := time.ParseDuration(
		os.Getenv("JWT_ACCESS_TOKEN_TTL"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid JWT_ACCESS_TOKEN_TTL: %w",
			err,
		)
	}

	cfg := &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		AppPort:          getEnv("APP_PORT", "8080"),

		DatabaseHost:     getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:     os.Getenv("DATABASE_USER"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:     os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		JWTIssuer:         os.Getenv("JWT_ISSUER"),
		JWTAccessTokenTTL: jwtAccessTokenTTL,
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

func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			c.DatabaseUser,
			c.DatabasePassword,
		),
		Host: fmt.Sprintf(
			"%s:%s",
			c.DatabaseHost,
			c.DatabasePort,
		),
		Path: c.DatabaseName,
	}

	query := u.Query()
	query.Set("sslmode", c.DatabaseSSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}