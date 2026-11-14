package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Promotion service.
type Config struct {
	AppEnv               string
	HTTPPort             string
	GRPCPort             string
	DatabaseHost         string
	DatabasePort         string
	DatabaseUser         string
	DatabasePassword     string
	DatabaseName         string
	DatabaseSSLMode      string
	TestDatabaseURL      string
	KafkaBrokers         []string
	PromotionEventsTopic string
	ServiceAuthName      string
	ServiceAuthSecret    string
}

// Load loads Promotion configuration. Existing environment variables take
// precedence over values in the service-local .env file.
func Load() (*Config, error) {
	_ = godotenv.Load("services/promotion/.env")

	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("HTTP_PORT", "8096"),
		GRPCPort:             getEnv("GRPC_PORT", "50057"),
		DatabaseHost:         getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:         getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:         strings.TrimSpace(os.Getenv("DATABASE_USER")),
		DatabasePassword:     os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:         strings.TrimSpace(os.Getenv("DATABASE_NAME")),
		DatabaseSSLMode:      getEnv("DATABASE_SSL_MODE", "disable"),
		TestDatabaseURL:      strings.TrimSpace(os.Getenv("PROMOTION_TEST_DATABASE_URL")),
		KafkaBrokers:         splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		PromotionEventsTopic: strings.TrimSpace(getEnv("PROMOTION_EVENTS_TOPIC", "promotion.events")),
		ServiceAuthName:      strings.TrimSpace(getEnv("SERVICE_AUTH_NAME", "promotion")),
		ServiceAuthSecret:    os.Getenv("SERVICE_AUTH_SECRET"),
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
	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}
	if cfg.PromotionEventsTopic == "" {
		return nil, fmt.Errorf("PROMOTION_EVENTS_TOPIC is required")
	}

	return cfg, nil
}

// DatabaseURL builds a PostgreSQL URL with escaped credentials and database name.
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

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
