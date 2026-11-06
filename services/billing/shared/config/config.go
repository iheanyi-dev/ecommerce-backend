package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains Billing service configuration.
type Config struct {
	DatabaseURL          string
	KafkaBrokers         []string
	PaymentEventsTopic   string
	StoreEventsTopic     string
	PaymentConsumerGroup string
	GRPCPort             string
	PaymentServiceAddr   string
	ServiceAuthName      string
	ServiceAuthSecret    string
}

// Load loads Billing configuration. Environment variables take precedence.
func Load() (*Config, error) {
	_ = godotenv.Load("services/billing/.env")

	cfg := &Config{
		DatabaseURL:          strings.TrimSpace(os.Getenv("BILLING_DATABASE_URL")),
		KafkaBrokers:         splitCSV(os.Getenv("KAFKA_BROKERS")),
		PaymentEventsTopic:   strings.TrimSpace(os.Getenv("PAYMENT_EVENTS_TOPIC")),
		StoreEventsTopic:     strings.TrimSpace(os.Getenv("BILLING_STORE_EVENTS_TOPIC")),
		PaymentConsumerGroup: strings.TrimSpace(os.Getenv("BILLING_PAYMENT_CONSUMER_GROUP")),
		GRPCPort:             strings.TrimSpace(os.Getenv("BILLING_GRPC_PORT")),
		PaymentServiceAddr:   strings.TrimSpace(getEnv("PAYMENT_SERVICE_ADDR", "localhost:9095")),
		ServiceAuthName:      strings.TrimSpace(os.Getenv("SERVICE_AUTH_NAME")),
		ServiceAuthSecret:    strings.TrimSpace(os.Getenv("SERVICE_AUTH_SECRET")),
	}

	switch {
	case cfg.DatabaseURL == "":
		return nil, fmt.Errorf("BILLING_DATABASE_URL is required")
	case len(cfg.KafkaBrokers) == 0:
		return nil, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	case cfg.PaymentEventsTopic == "":
		return nil, fmt.Errorf("PAYMENT_EVENTS_TOPIC is required")
	case cfg.StoreEventsTopic == "":
		return nil, fmt.Errorf("BILLING_STORE_EVENTS_TOPIC is required")
	case cfg.PaymentConsumerGroup == "":
		return nil, fmt.Errorf("BILLING_PAYMENT_CONSUMER_GROUP is required")
	case cfg.GRPCPort == "":
		return nil, fmt.Errorf("BILLING_GRPC_PORT is required")
	case cfg.PaymentServiceAddr == "":
		return nil, fmt.Errorf("PAYMENT_SERVICE_ADDR is required")
	case cfg.ServiceAuthName == "" || cfg.ServiceAuthSecret == "":
		return nil, fmt.Errorf("SERVICE_AUTH_NAME and SERVICE_AUTH_SECRET are required")
	}

	return cfg, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

// getEnv returns the environment value or its fallback.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
