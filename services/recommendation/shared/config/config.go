package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config contains the runtime configuration for the Recommendation service.
type Config struct {
	ServiceName   string
	ServiceSecret string

	HTTPAddr string

	EmbeddingServiceAddr   string
	EmbeddingServiceName   string
	EmbeddingServiceSecret string
	EmbeddingGRPCTimeout   time.Duration
}

// Load loads Recommendation configuration from environment variables.
//
// The service-local .env file is loaded as a local-development convenience.
// Docker, CI, and production environments can provide configuration directly.
func Load() (*Config, error) {
	_ = godotenv.Load("services/recommendation/.env")

	timeoutSeconds, err := strconv.ParseFloat(
		getEnv("EMBEDDING_GRPC_TIMEOUT_SECONDS", "10"),
		64,
	)
	if err != nil || timeoutSeconds <= 0 {
		return nil, fmt.Errorf(
			"EMBEDDING_GRPC_TIMEOUT_SECONDS must be a positive number",
		)
	}

	cfg := &Config{
		ServiceName:            getEnv("RECOMMENDATION_SERVICE_NAME", "recommendation"),
		ServiceSecret:          os.Getenv("RECOMMENDATION_SERVICE_SECRET"),
		HTTPAddr:               getEnv("RECOMMENDATION_HTTP_ADDR", "0.0.0.0:8091"),
		EmbeddingServiceAddr:   getEnv("EMBEDDING_SERVICE_ADDR", "localhost:8092"),
		EmbeddingServiceName:   getEnv("EMBEDDING_SERVICE_NAME", "chat"),
		EmbeddingServiceSecret: os.Getenv("EMBEDDING_SERVICE_SECRET"),
		EmbeddingGRPCTimeout:   time.Duration(timeoutSeconds * float64(time.Second)),
	}

	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("RECOMMENDATION_SERVICE_NAME must not be empty")
	}
	if cfg.ServiceSecret == "" {
		return nil, fmt.Errorf("RECOMMENDATION_SERVICE_SECRET is required")
	}
	if _, _, err := net.SplitHostPort(cfg.HTTPAddr); err != nil {
		return nil, fmt.Errorf("RECOMMENDATION_HTTP_ADDR must use host:port format: %w", err)
	}
	if cfg.EmbeddingServiceAddr == "" {
		return nil, fmt.Errorf("EMBEDDING_SERVICE_ADDR must not be empty")
	}
	if cfg.EmbeddingServiceName == "" {
		return nil, fmt.Errorf("EMBEDDING_SERVICE_NAME must not be empty")
	}
	if cfg.EmbeddingServiceSecret == "" {
		return nil, fmt.Errorf("EMBEDDING_SERVICE_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
