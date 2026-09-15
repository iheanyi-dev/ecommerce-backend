package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains configuration required by the API Gateway.
//
// Gateway configuration is intentionally limited to concerns owned by the
// gateway itself and the addresses and credentials required to communicate
// with internal services.
type Config struct {
	AppEnv  string
	AppPort string

	// IdentityServiceURL is the base URL used by the gateway when forwarding
	// requests to the Identity service.
	IdentityServiceURL string

	// IdentityServiceName is the service identity presented to Identity.
	IdentityServiceName string

	// IdentityServiceSecret is the shared secret used for Gateway-to-Identity
	// service authentication.
	IdentityServiceSecret string
}

// Load loads Gateway configuration from environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load("gateway/.env")

	identityServiceURL := getEnv(
		"IDENTITY_SERVICE_URL",
		"http://localhost:8080",
	)

	if err := validateServiceURL(
		"IDENTITY_SERVICE_URL",
		identityServiceURL,
	); err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:                getEnv("APP_ENV", "development"),
		AppPort:               getEnv("APP_PORT", "8081"),
		IdentityServiceURL:    identityServiceURL,
		IdentityServiceName:   os.Getenv("IDENTITY_SERVICE_NAME"),
		IdentityServiceSecret: os.Getenv("IDENTITY_SERVICE_SECRET"),
	}

	if cfg.IdentityServiceName == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_NAME is required")
	}

	if cfg.IdentityServiceSecret == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_SECRET is required")
	}

	return cfg, nil
}

func validateServiceURL(name string, value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf(
			"%s must be a valid URL: %w",
			name,
			err,
		)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf(
			"%s must use http or https scheme",
			name,
		)
	}

	if strings.TrimSpace(parsedURL.Host) == "" {
		return fmt.Errorf(
			"%s must contain a host",
			name,
		)
	}

	return nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
