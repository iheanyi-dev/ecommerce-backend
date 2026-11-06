package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/shared/config"
)

func TestLoad(t *testing.T) {
	t.Setenv("ORDER_SERVICE_AUTH_NAME", "order")
	t.Setenv("ORDER_SERVICE_AUTH_SECRET", "order-secret")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("IDENTITY_SERVICE_URL", "http://identity:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "gateway-secret")
	t.Setenv("STORE_SERVICE_URL", "http://store:8082")
	t.Setenv("STORE_SERVICE_NAME", "gateway")
	t.Setenv("STORE_SERVICE_SECRET", "store-secret")
	t.Setenv("STORAGE_SERVICE_URL", "http://storage:8083")
	t.Setenv("STORAGE_SERVICE_NAME", "gateway")
	t.Setenv("STORAGE_SERVICE_SECRET", "storage-secret")
	t.Setenv("RECOMMENDATION_SERVICE_NAME", "recommendation")
	t.Setenv("RECOMMENDATION_SERVICE_SECRET", "recommendation-secret")
	t.Setenv("PRODUCT_SERVICE_NAME", "product")
	t.Setenv("PRODUCT_SERVICE_SECRET", "product-secret")
	t.Setenv("CART_SERVICE_NAME", "cart")
	t.Setenv("CART_SERVICE_SECRET", "cart-secret")
	t.Setenv("JWT_SECRET", "this-is-a-valid-development-jwt-secret-123456")
	t.Setenv("JWT_ISSUER", "identity-service")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "test", cfg.AppEnv)
	require.Equal(t, "9090", cfg.AppPort)
	require.Equal(t, "http://identity:8080", cfg.IdentityServiceURL)
	require.Equal(t, "gateway", cfg.IdentityServiceName)
	require.Equal(t, "gateway-secret", cfg.IdentityServiceSecret)
	require.Equal(t, "this-is-a-valid-development-jwt-secret-123456", cfg.JWTSecret)
	require.Equal(t, "identity-service", cfg.JWTIssuer)
}

func TestLoad_UsesDefaults(t *testing.T) {
	t.Setenv("ORDER_SERVICE_AUTH_NAME", "order")
	t.Setenv("ORDER_SERVICE_AUTH_SECRET", "order-secret")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("IDENTITY_SERVICE_URL", "")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "gateway-secret")
	t.Setenv("STORE_SERVICE_URL", "http://store:8082")
	t.Setenv("STORE_SERVICE_NAME", "gateway")
	t.Setenv("STORE_SERVICE_SECRET", "store-secret")
	t.Setenv("STORAGE_SERVICE_URL", "http://storage:8083")
	t.Setenv("STORAGE_SERVICE_NAME", "gateway")
	t.Setenv("STORAGE_SERVICE_SECRET", "storage-secret")
	t.Setenv("RECOMMENDATION_SERVICE_NAME", "recommendation")
	t.Setenv("RECOMMENDATION_SERVICE_SECRET", "recommendation-secret")
	t.Setenv("PRODUCT_SERVICE_NAME", "product")
	t.Setenv("PRODUCT_SERVICE_SECRET", "product-secret")
	t.Setenv("CART_SERVICE_NAME", "cart")
	t.Setenv("CART_SERVICE_SECRET", "cart-secret")
	t.Setenv("JWT_SECRET", "this-is-a-valid-development-jwt-secret-123456")
	t.Setenv("JWT_ISSUER", "identity-service")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "development", cfg.AppEnv)
	require.Equal(t, "8081", cfg.AppPort)
	require.Equal(t, "http://localhost:8080", cfg.IdentityServiceURL)
	require.Equal(t, "gateway", cfg.IdentityServiceName)
	require.Equal(t, "gateway-secret", cfg.IdentityServiceSecret)
	require.Equal(t, "this-is-a-valid-development-jwt-secret-123456", cfg.JWTSecret)
	require.Equal(t, "identity-service", cfg.JWTIssuer)
}

func TestLoad_RejectsInvalidIdentityServiceURL(t *testing.T) {
	t.Setenv("ORDER_SERVICE_AUTH_NAME", "order")
	t.Setenv("ORDER_SERVICE_AUTH_SECRET", "order-secret")
	t.Setenv("IDENTITY_SERVICE_URL", "://invalid-url")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "gateway-secret")
	t.Setenv("STORE_SERVICE_URL", "http://store:8082")
	t.Setenv("STORE_SERVICE_NAME", "gateway")
	t.Setenv("STORE_SERVICE_SECRET", "store-secret")
	t.Setenv("STORAGE_SERVICE_URL", "http://storage:8083")
	t.Setenv("STORAGE_SERVICE_NAME", "gateway")
	t.Setenv("STORAGE_SERVICE_SECRET", "storage-secret")
	t.Setenv("RECOMMENDATION_SERVICE_NAME", "recommendation")
	t.Setenv("RECOMMENDATION_SERVICE_SECRET", "recommendation-secret")
	t.Setenv("PRODUCT_SERVICE_NAME", "product")
	t.Setenv("PRODUCT_SERVICE_SECRET", "product-secret")
	t.Setenv("CART_SERVICE_NAME", "cart")
	t.Setenv("CART_SERVICE_SECRET", "cart-secret")
	t.Setenv("JWT_SECRET", "this-is-a-valid-development-jwt-secret-123456")
	t.Setenv("JWT_ISSUER", "identity-service")

	_, err := config.Load()

	require.Error(t, err)
}

func TestLoad_RejectsMissingIdentityServiceName(t *testing.T) {
	t.Setenv("ORDER_SERVICE_AUTH_NAME", "order")
	t.Setenv("ORDER_SERVICE_AUTH_SECRET", "order-secret")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "")
	t.Setenv("IDENTITY_SERVICE_SECRET", "gateway-secret")
	t.Setenv("STORE_SERVICE_URL", "http://store:8082")
	t.Setenv("STORE_SERVICE_NAME", "gateway")
	t.Setenv("STORE_SERVICE_SECRET", "store-secret")
	t.Setenv("STORAGE_SERVICE_URL", "http://storage:8083")
	t.Setenv("STORAGE_SERVICE_NAME", "gateway")
	t.Setenv("STORAGE_SERVICE_SECRET", "storage-secret")
	t.Setenv("RECOMMENDATION_SERVICE_NAME", "recommendation")
	t.Setenv("RECOMMENDATION_SERVICE_SECRET", "recommendation-secret")
	t.Setenv("PRODUCT_SERVICE_NAME", "product")
	t.Setenv("PRODUCT_SERVICE_SECRET", "product-secret")
	t.Setenv("CART_SERVICE_NAME", "cart")
	t.Setenv("CART_SERVICE_SECRET", "cart-secret")
	t.Setenv("JWT_SECRET", "this-is-a-valid-development-jwt-secret-123456")
	t.Setenv("JWT_ISSUER", "identity-service")

	_, err := config.Load()

	require.Error(t, err)
}

func TestLoad_RejectsMissingIdentityServiceSecret(t *testing.T) {
	t.Setenv("ORDER_SERVICE_AUTH_NAME", "order")
	t.Setenv("ORDER_SERVICE_AUTH_SECRET", "order-secret")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "")
	t.Setenv("JWT_SECRET", "this-is-a-valid-development-jwt-secret-123456")
	t.Setenv("JWT_ISSUER", "identity-service")

	_, err := config.Load()

	require.Error(t, err)
}
