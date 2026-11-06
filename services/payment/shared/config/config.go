package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains configuration owned by the Payment service.
type Config struct {
	AppEnv   string
	HTTPPort string
	GRPCPort string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
	TestDatabaseURL  string

	PaystackSecretKey string
	PaystackBaseURL   string

	KafkaBrokers       []string
	PaymentEventsTopic string

	ServiceAuthName          string
	ServiceAuthSecret        string
	BeneficiaryEncryptionKey []byte
}

// Load loads Payment configuration.
//
// Environment variables supplied by Docker, CI, or production take precedence
// over values loaded from the service-local .env file.
func Load() (*Config, error) {
	_ = godotenv.Load("services/payment/.env")

	encryptionKey, err := hex.DecodeString(
		strings.TrimSpace(os.Getenv("PAYMENT_BENEFICIARY_ENCRYPTION_KEY")),
	)
	if err != nil || len(encryptionKey) != 32 {
		return nil, fmt.Errorf("PAYMENT_BENEFICIARY_ENCRYPTION_KEY must be a valid 64-character hexadecimal AES-256 key")
	}

	cfg := &Config{
		AppEnv:                   getEnv("APP_ENV", "development"),
		HTTPPort:                 getEnv("PAYMENT_HTTP_PORT", "8095"),
		GRPCPort:                 getEnv("PAYMENT_GRPC_PORT", "9095"),
		DatabaseHost:             getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:             getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:             strings.TrimSpace(os.Getenv("DATABASE_USER")),
		DatabasePassword:         os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:             strings.TrimSpace(os.Getenv("DATABASE_NAME")),
		DatabaseSSLMode:          getEnv("DATABASE_SSL_MODE", "disable"),
		TestDatabaseURL:          strings.TrimSpace(os.Getenv("PAYMENT_TEST_DATABASE_URL")),
		PaystackSecretKey:        strings.TrimSpace(os.Getenv("PAYSTACK_SECRET_KEY")),
		PaystackBaseURL:          getEnv("PAYSTACK_BASE_URL", "https://api.paystack.co"),
		KafkaBrokers:             splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		PaymentEventsTopic:       strings.TrimSpace(getEnv("PAYMENT_EVENTS_TOPIC", "payment.events")),
		ServiceAuthName:          strings.TrimSpace(os.Getenv("SERVICE_AUTH_NAME")),
		ServiceAuthSecret:        os.Getenv("SERVICE_AUTH_SECRET"),
		BeneficiaryEncryptionKey: encryptionKey,
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

	if cfg.PaystackSecretKey == "" {
		return nil, fmt.Errorf("PAYSTACK_SECRET_KEY is required")
	}

	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}

	if cfg.PaymentEventsTopic == "" {
		return nil, fmt.Errorf("PAYMENT_EVENTS_TOPIC is required")
	}

	if cfg.ServiceAuthName == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_NAME is required")
	}

	if cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_SECRET is required")
	}

	if err := validatePaystackBaseURL(cfg.PaystackBaseURL); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DatabaseURL builds the PostgreSQL connection URL with correctly escaped
// credentials and database names.
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

func validatePaystackBaseURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("PAYSTACK_BASE_URL is invalid: %w", err)
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("PAYSTACK_BASE_URL must use HTTP or HTTPS")
	}

	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("PAYSTACK_BASE_URL must be an absolute API URL without credentials, query, or fragment")
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// splitCSV parses a comma-separated environment variable into trimmed,
// non-empty values.
func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}

	return values
}
