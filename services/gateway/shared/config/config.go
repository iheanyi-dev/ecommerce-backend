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

	// StoreServiceURL is the base URL used by the Gateway when forwarding
	// Store API requests.
	StoreServiceURL string

	// StoreServiceName is the service identity presented to Store.
	StoreServiceName string

	// StoreServiceSecret is the shared secret used for Gateway-to-Store
	// service authentication.
	StoreServiceSecret string

	// StorageServiceURL is the base URL used by the Gateway when forwarding
	// Storage public-object requests.
	StorageServiceURL string

	// StorageServiceName is the service identity presented to Storage.
	StorageServiceName string

	// StorageServiceSecret is the shared secret used for Gateway-to-Storage
	// service authentication.
	StorageServiceSecret string

	// RecommendationServiceURL is the base URL used by the Gateway when
	// forwarding Recommendation API requests.
	RecommendationServiceURL string

	// RecommendationServiceName is the service identity presented to
	// Recommendation.
	RecommendationServiceName string

	// RecommendationServiceSecret is the shared secret used for
	// Gateway-to-Recommendation service authentication.
	RecommendationServiceSecret string

	// Product service connection and service-authentication credentials.
	ProductServiceURL    string
	ProductServiceName   string
	ProductServiceSecret string

	// Cart service connection and service-authentication credentials.
	CartServiceURL    string
	CartServiceName   string
	CartServiceSecret string

	// Order service connection and service-authentication credentials.
	OrderServiceURL        string
	OrderServiceAuthName   string
	OrderServiceAuthSecret string

	// Payment service connection. Gateway access is restricted to
	// Payment routes that require authenticated user identity.
	PaymentServiceURL string

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
	_ = godotenv.Load("services/gateway/.env")

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

	storeServiceURL := getEnv(
		"STORE_SERVICE_URL",
		"http://localhost:8082",
	)

	if err := validateServiceURL(
		"STORE_SERVICE_URL",
		storeServiceURL,
	); err != nil {
		return nil, err
	}

	storageServiceURL := getEnv(
		"STORAGE_SERVICE_URL",
		"http://localhost:8083",
	)

	if err := validateServiceURL(
		"STORAGE_SERVICE_URL",
		storageServiceURL,
	); err != nil {
		return nil, err
	}

	recommendationServiceURL := getEnv(
		"RECOMMENDATION_SERVICE_URL",
		"http://localhost:8091",
	)

	if err := validateServiceURL(
		"RECOMMENDATION_SERVICE_URL",
		recommendationServiceURL,
	); err != nil {
		return nil, err
	}

	productServiceURL := getEnv(
		"PRODUCT_SERVICE_URL",
		"http://localhost:8083",
	)

	if err := validateServiceURL("PRODUCT_SERVICE_URL", productServiceURL); err != nil {
		return nil, err
	}

	cartServiceURL := getEnv(
		"CART_SERVICE_URL",
		"http://localhost:8090",
	)

	if err := validateServiceURL("CART_SERVICE_URL", cartServiceURL); err != nil {
		return nil, err
	}

	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8094")
	if err := validateServiceURL("ORDER_SERVICE_URL", orderServiceURL); err != nil {
		return nil, err
	}

	paymentServiceURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8095")
	if err := validateServiceURL("PAYMENT_SERVICE_URL", paymentServiceURL); err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8081"),

		IdentityServiceURL:    identityServiceURL,
		IdentityServiceName:   os.Getenv("IDENTITY_SERVICE_NAME"),
		IdentityServiceSecret: os.Getenv("IDENTITY_SERVICE_SECRET"),

		StoreServiceURL:    storeServiceURL,
		StoreServiceName:   os.Getenv("STORE_SERVICE_NAME"),
		StoreServiceSecret: os.Getenv("STORE_SERVICE_SECRET"),

		StorageServiceURL:    storageServiceURL,
		StorageServiceName:   os.Getenv("STORAGE_SERVICE_NAME"),
		StorageServiceSecret: os.Getenv("STORAGE_SERVICE_SECRET"),

		RecommendationServiceURL:    recommendationServiceURL,
		RecommendationServiceName:   os.Getenv("RECOMMENDATION_SERVICE_NAME"),
		RecommendationServiceSecret: os.Getenv("RECOMMENDATION_SERVICE_SECRET"),
		ProductServiceURL:           productServiceURL,
		ProductServiceName:          os.Getenv("PRODUCT_SERVICE_NAME"),
		ProductServiceSecret:        os.Getenv("PRODUCT_SERVICE_SECRET"),
		CartServiceURL:              cartServiceURL,
		CartServiceName:             os.Getenv("CART_SERVICE_NAME"),
		CartServiceSecret:           os.Getenv("CART_SERVICE_SECRET"),
		OrderServiceURL:             orderServiceURL,
		OrderServiceAuthName:        getEnv("ORDER_SERVICE_AUTH_NAME", "order"),
		OrderServiceAuthSecret:      os.Getenv("ORDER_SERVICE_AUTH_SECRET"),
		PaymentServiceURL:           paymentServiceURL,

		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTIssuer: os.Getenv("JWT_ISSUER"),
	}

	if cfg.IdentityServiceName == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_NAME is required")
	}

	if cfg.IdentityServiceSecret == "" {
		return nil, fmt.Errorf("IDENTITY_SERVICE_SECRET is required")
	}

	if cfg.StoreServiceName == "" {
		return nil, fmt.Errorf("STORE_SERVICE_NAME is required")
	}

	if cfg.StoreServiceSecret == "" {
		return nil, fmt.Errorf("STORE_SERVICE_SECRET is required")
	}

	if cfg.StorageServiceName == "" {
		return nil, fmt.Errorf("STORAGE_SERVICE_NAME is required")
	}

	if cfg.StorageServiceSecret == "" {
		return nil, fmt.Errorf("STORAGE_SERVICE_SECRET is required")
	}

	if cfg.RecommendationServiceName == "" {
		return nil, fmt.Errorf("RECOMMENDATION_SERVICE_NAME is required")
	}

	if cfg.RecommendationServiceSecret == "" {
		return nil, fmt.Errorf("RECOMMENDATION_SERVICE_SECRET is required")
	}

	if cfg.ProductServiceName == "" {
		return nil, fmt.Errorf("PRODUCT_SERVICE_NAME is required")
	}

	if cfg.ProductServiceSecret == "" {
		return nil, fmt.Errorf("PRODUCT_SERVICE_SECRET is required")
	}

	if cfg.CartServiceName == "" {
		return nil, fmt.Errorf("CART_SERVICE_NAME is required")
	}

	if cfg.CartServiceSecret == "" {
		return nil, fmt.Errorf("CART_SERVICE_SECRET is required")
	}

	if strings.TrimSpace(cfg.OrderServiceAuthName) == "" {
		return nil, fmt.Errorf("ORDER_SERVICE_AUTH_NAME is required")
	}

	if cfg.OrderServiceAuthSecret == "" {
		return nil, fmt.Errorf("ORDER_SERVICE_AUTH_SECRET is required")
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
