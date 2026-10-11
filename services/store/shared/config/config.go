package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Store service.
//
// Store does not contain JWT configuration. User authentication is performed
// by the Gateway, while Store consumes the trusted identity propagated by the
// Gateway. ServiceAuth fields authenticate trusted internal callers.
type Config struct {
	AppEnv  string
	AppPort string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string

	ServiceAuthName             string
	ServiceAuthSecret           string
	IdentityServiceAddr         string
	StorageServiceAddr          string
	RecommendationServiceAddr   string
	RecommendationServiceName   string
	RecommendationServiceSecret string
	ChatServiceAddr             string
	ChatServiceName             string
	ChatServiceSecret           string
	ProductServiceAddr          string
	ProductServiceName          string
	ProductServiceSecret        string
	KafkaBrokers                string
}

// Load loads Store configuration from environment variables.
//
// The .env file is loaded only as a local-development convenience. Docker,
// CI, and production environments can provide configuration directly through
// environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load("services/store/.env")

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8082"),

		DatabaseHost:     getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:     os.Getenv("DATABASE_USER"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:     os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:  getEnv("DATABASE_SSL_MODE", "disable"),

		ServiceAuthName:             os.Getenv("SERVICE_AUTH_NAME"),
		ServiceAuthSecret:           os.Getenv("SERVICE_AUTH_SECRET"),
		IdentityServiceAddr:         getEnv("IDENTITY_SERVICE_ADDR", "localhost:9090"),
		StorageServiceAddr:          getEnv("STORAGE_SERVICE_ADDR", "localhost:9082"),
		RecommendationServiceAddr:   getEnv("RECOMMENDATION_SERVICE_ADDR", "localhost:8090"),
		RecommendationServiceName:   getEnv("RECOMMENDATION_SERVICE_NAME", "recommendation"),
		RecommendationServiceSecret: os.Getenv("RECOMMENDATION_SERVICE_SECRET"),
		ChatServiceAddr:             getEnv("CHAT_SERVICE_ADDR", "localhost:9092"),
		ChatServiceName:             getEnv("CHAT_SERVICE_NAME", "chat"),
		ChatServiceSecret:           os.Getenv("CHAT_SERVICE_SECRET"),
		ProductServiceAddr:          getEnv("PRODUCT_SERVICE_ADDR", "localhost:9091"),
		ProductServiceName:          getEnv("PRODUCT_SERVICE_NAME", "product"),
		ProductServiceSecret:        os.Getenv("PRODUCT_SERVICE_SECRET"),
		KafkaBrokers:                getEnv("KAFKA_BROKERS", "localhost:19092"),
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

	if cfg.ServiceAuthName == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_NAME is required")
	}

	if cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_SECRET is required")
	}

	if cfg.RecommendationServiceSecret == "" {
		return nil, fmt.Errorf("RECOMMENDATION_SERVICE_SECRET is required")
	}

	return cfg, nil
}

// DatabaseURL returns the PostgreSQL connection URL used by the Store
// persistence infrastructure.
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
