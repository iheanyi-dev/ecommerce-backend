package config_test

import (
	"os"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8093")
	t.Setenv("GRPC_PORT", "9093")
	t.Setenv("STORAGE_ROOT", "./test-storage")
	t.Setenv("SERVICE_AUTH_NAME", "store")
	t.Setenv("SERVICE_AUTH_SECRET", "storage-secret")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "test", cfg.AppEnv)
	require.Equal(t, "8093", cfg.AppPort)
	require.Equal(t, "9093", cfg.GRPCPort)
	require.Equal(t, "./test-storage", cfg.StorageRoot)
	require.Equal(t, "store", cfg.ServiceAuthName)
	require.Equal(t, "storage-secret", cfg.ServiceAuthSecret)
}

func TestLoad_DefaultsGRPCPort(t *testing.T) {
	setRequiredEnvironment(t)
	_ = os.Unsetenv("GRPC_PORT")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "9082", cfg.GRPCPort)
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

func setRequiredEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("STORAGE_ROOT", "./storage-data")
	t.Setenv("SERVICE_AUTH_NAME", "store")
	t.Setenv("SERVICE_AUTH_SECRET", "storage-secret")

	for _, key := range []string{
		"APP_ENV",
		"APP_PORT",
		"GRPC_PORT",
	} {
		_ = os.Unsetenv(key)
	}
}
