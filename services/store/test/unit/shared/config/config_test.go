package config_test

import (
	"os"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8090")

	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "store")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_NAME", "store")
	t.Setenv("DATABASE_SSL_MODE", "disable")

	t.Setenv("SERVICE_AUTH_NAME", "gateway")
	t.Setenv("SERVICE_AUTH_SECRET", "store-secret")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "test", cfg.AppEnv)
	require.Equal(t, "8090", cfg.AppPort)
	require.Equal(t, "localhost", cfg.DatabaseHost)
	require.Equal(t, "5432", cfg.DatabasePort)
	require.Equal(t, "store", cfg.DatabaseUser)
	require.Equal(t, "secret", cfg.DatabasePassword)
	require.Equal(t, "store", cfg.DatabaseName)
	require.Equal(t, "disable", cfg.DatabaseSSLMode)
	require.Equal(t, "gateway", cfg.ServiceAuthName)
	require.Equal(t, "store-secret", cfg.ServiceAuthSecret)
}

func TestLoad_RequiresDatabaseUser(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DATABASE_USER", "")

	_, err := config.Load()

	require.EqualError(t, err, "DATABASE_USER is required")
}

func TestLoad_RequiresDatabasePassword(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DATABASE_PASSWORD", "")

	_, err := config.Load()

	require.EqualError(t, err, "DATABASE_PASSWORD is required")
}

func TestLoad_RequiresDatabaseName(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DATABASE_NAME", "")

	_, err := config.Load()

	require.EqualError(t, err, "DATABASE_NAME is required")
}

func TestLoad_RequiresServiceAuthName(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("SERVICE_AUTH_NAME", "")

	_, err := config.Load()

	require.EqualError(t, err, "SERVICE_AUTH_NAME is required")
}

func TestLoad_RequiresServiceAuthSecret(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("SERVICE_AUTH_SECRET", "")

	_, err := config.Load()

	require.EqualError(t, err, "SERVICE_AUTH_SECRET is required")
}

func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "store")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_NAME", "store")
	t.Setenv("DATABASE_SSL_MODE", "disable")
	t.Setenv("SERVICE_AUTH_NAME", "gateway")
	t.Setenv("SERVICE_AUTH_SECRET", "secret")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(
		t,
		"postgres://store:secret@localhost:5432/store?sslmode=disable",
		cfg.DatabaseURL(),
	)
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("DATABASE_USER", "store")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_NAME", "store")
	t.Setenv("SERVICE_AUTH_NAME", "gateway")
	t.Setenv("SERVICE_AUTH_SECRET", "secret")

	// Prevent a developer's local .env from influencing the test.
	for _, key := range []string{
		"APP_ENV",
		"APP_PORT",
		"DATABASE_HOST",
		"DATABASE_PORT",
		"DATABASE_SSL_MODE",
	} {
		_ = os.Unsetenv(key)
	}
}
