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
// Gateway configuration is intentionally limited to concerns owned by
// the Gateway itself and the addresses/credentials required to communicate
// with internal services.
//
// The Gateway also needs the Identity Service's JWT verification
// configuration because the Gateway is the external authentication boundary
// for client-facing business-service requests.
type Config struct {
	AppEnv  string
	AppPort string

	// IdentityServiceURL is the base URL used by the Gateway when forwarding
	// requests to the Identity service.
	IdentityServiceURL string

	// IdentityServiceName is the service identity presented to Identity.
	IdentityServiceName string

	// IdentityServiceSecret is the shared secret used for Gateway-to-Identity
	// service authentication.
	IdentityServiceSecret string

	// JWTSecret is the secret used by Identity to sign access tokens.
	//
	// The Gateway uses this secret only to validate client access tokens.
	// It never issues tokens.
	JWTSecret string

	// JWTIssuer identifies the service that issued valid access tokens.
	JWTIssuer string
}

// Load loads Gateway configuration from environment variables.
func Load() (*Config, error) {
	// Load .env for local development.
	//
	// In Docker/production, environment variables can be supplied directly
	// by the runtime.
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
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8081"),

		IdentityServiceURL:    identityServiceURL,
		IdentityServiceName:   os.Getenv("IDENTITY_SERVICE_NAME"),
		IdentityServiceSecret: os.Getenv("IDENTITY_SERVICE_SECRET"),

		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTIssuer: os.Getenv("JWT_ISSUER"),
	}

	if cfg.IdentityServiceName == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_NAME is required")
	}

	if cfg.IdentityServiceSecret == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_SECRET is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf(
			"JWT_SECRET must be at least 32 characters",
		)
	}

	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		return nil, fmt.Errorf("JWT_ISSUER is required")
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
