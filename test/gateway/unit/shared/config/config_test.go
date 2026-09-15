package config_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "test-secret")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "test", cfg.AppEnv)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "http://localhost:8080", cfg.IdentityServiceURL)
	assert.Equal(t, "gateway", cfg.IdentityServiceName)
	assert.Equal(t, "test-secret", cfg.IdentityServiceSecret)
}

func TestLoad_UsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("IDENTITY_SERVICE_URL", "")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "test-secret")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "http://localhost:8080", cfg.IdentityServiceURL)
	assert.Equal(t, "gateway", cfg.IdentityServiceName)
	assert.Equal(t, "test-secret", cfg.IdentityServiceSecret)
}

func TestLoad_RejectsInvalidIdentityServiceURL(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("IDENTITY_SERVICE_URL", "identity-service")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "test-secret")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorContains(
		t,
		err,
		"IDENTITY_SERVICE_URL",
	)
}

func TestLoad_RejectsMissingIdentityServiceName(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "")
	t.Setenv("IDENTITY_SERVICE_SECRET", "test-secret")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorContains(
		t,
		err,
		"IDENTITY_SERVICE_NAME",
	)
}

func TestLoad_RejectsMissingIdentityServiceSecret(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")
	t.Setenv("IDENTITY_SERVICE_NAME", "gateway")
	t.Setenv("IDENTITY_SERVICE_SECRET", "")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorContains(
		t,
		err,
		"IDENTITY_SERVICE_SECRET",
	)
}
