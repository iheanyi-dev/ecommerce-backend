package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Product service.
type Config struct {
	AppEnv                 string
	AppPort                string
	GRPCPort               string
	GRPCServiceAuthName    string
	GRPCServiceAuthSecret  string
	DatabaseHost           string
	DatabasePort           string
	DatabaseUser           string
	DatabasePassword       string
	DatabaseName           string
	DatabaseSSLMode        string
	StorageServiceAddr     string
	StoreServiceAddr       string
	KafkaBrokers           string
	PromotionEventsTopic   string
	PromotionConsumerGroup string
	ServiceAuthName        string
	ServiceAuthSecret      string
}

// ErrMissingAppPort indicates the HTTP listener port was not configured.
var ErrMissingAppPort = fmt.Errorf("APP_PORT is required")

// Load loads Product configuration. The .env file is a local-development
// convenience; deployed environments may provide variables directly.
func Load() (*Config, error) {
	_ = godotenv.Load("services/product/.env")

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppPort:                getEnv("APP_PORT", "8083"),
		GRPCPort:               getEnv("GRPC_PORT", "9091"),
		GRPCServiceAuthName:    getEnv("GRPC_SERVICE_AUTH_NAME", "store"),
		GRPCServiceAuthSecret:  os.Getenv("GRPC_SERVICE_AUTH_SECRET"),
		DatabaseHost:           getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:           getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:           os.Getenv("DATABASE_USER"),
		DatabasePassword:       os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:           os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:        getEnv("DATABASE_SSL_MODE", "disable"),
		StorageServiceAddr:     os.Getenv("STORAGE_SERVICE_ADDR"),
		StoreServiceAddr:       os.Getenv("STORE_SERVICE_ADDR"),
		KafkaBrokers:           getEnv("KAFKA_BROKERS", "localhost:19092"),
		PromotionEventsTopic:   getEnv("PROMOTION_EVENTS_TOPIC", "promotion.events"),
		PromotionConsumerGroup: getEnv("PRODUCT_PROMOTION_CONSUMER_GROUP", "product-promotion-consumer"),
		ServiceAuthName:        os.Getenv("SERVICE_AUTH_NAME"),
		ServiceAuthSecret:      os.Getenv("SERVICE_AUTH_SECRET"),
	}

	if cfg.PromotionEventsTopic == "" || cfg.PromotionConsumerGroup == "" {
		return nil, fmt.Errorf("PROMOTION_EVENTS_TOPIC and PRODUCT_PROMOTION_CONSUMER_GROUP are required")
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
	if cfg.StorageServiceAddr == "" {
		return nil, fmt.Errorf("STORAGE_SERVICE_ADDR is required")
	}
	if cfg.StoreServiceAddr == "" {
		return nil, fmt.Errorf("STORE_SERVICE_ADDR is required")
	}
	if cfg.ServiceAuthName == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_NAME is required")
	}
	if cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_SECRET is required")
	}
	if cfg.GRPCPort == "" {
		return nil, fmt.Errorf("GRPC_PORT is required")
	}
	if cfg.GRPCServiceAuthName == "" {
		return nil, fmt.Errorf("GRPC_SERVICE_AUTH_NAME is required")
	}
	if cfg.GRPCServiceAuthSecret == "" {
		return nil, fmt.Errorf("GRPC_SERVICE_AUTH_SECRET is required")
	}

	return cfg, nil
}

// DatabaseURL returns the PostgreSQL connection URL.
func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DatabaseUser, c.DatabasePassword),
		Host:   fmt.Sprintf("%s:%s", c.DatabaseHost, c.DatabasePort),
		Path:   c.DatabaseName,
	}
	query := u.Query()
	query.Set("sslmode", c.DatabaseSSLMode)
	u.RawQuery = query.Encode()
	return u.String()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
