package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

func TestLoad_ReturnsConfigurationFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "test_user")
	t.Setenv("DATABASE_PASSWORD", "test_password")
	t.Setenv("DATABASE_NAME", "identity_test")
	t.Setenv("DATABASE_SSLMODE", "disable")

	cfg, err := config.Load()

	require.NoError(t, err)

	assert.Equal(t, "test", cfg.AppEnv)
	assert.Equal(t, "9000", cfg.AppPort)
	assert.Equal(t, "localhost", cfg.DatabaseHost)
	assert.Equal(t, "5432", cfg.DatabasePort)
	assert.Equal(t, "test_user", cfg.DatabaseUser)
	assert.Equal(t, "test_password", cfg.DatabasePassword)
	assert.Equal(t, "identity_test", cfg.DatabaseName)
	assert.Equal(t, "disable", cfg.DatabaseSSLMode)
}

func TestLoad_ReturnsErrorWhenDatabaseUserIsMissing(t *testing.T) {
	t.Setenv("DATABASE_USER", "")
	t.Setenv("DATABASE_PASSWORD", "password")
	t.Setenv("DATABASE_NAME", "identity")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_ReturnsErrorWhenDatabasePasswordIsMissing(t *testing.T) {
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "")
	t.Setenv("DATABASE_NAME", "identity")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_ReturnsErrorWhenDatabaseNameIsMissing(t *testing.T) {
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "password")
	t.Setenv("DATABASE_NAME", "")

	cfg, err := config.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "password")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "identity")
	t.Setenv("DATABASE_SSLMODE", "disable")

	cfg, err := config.Load()

	require.NoError(t, err)

	assert.Equal(
		t,
		"postgres://postgres:password@localhost:5432/identity?sslmode=disable",
		cfg.DatabaseURL(),
	)
}

func TestDatabaseURL_EscapesSpecialCharactersInPassword(t *testing.T) {
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "my@password:123")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "identity")
	t.Setenv("DATABASE_SSLMODE", "disable")

	cfg, err := config.Load()

	require.NoError(t, err)

	assert.Equal(
		t,
		"postgres://postgres:my%40password%3A123@localhost:5432/identity?sslmode=disable",
		cfg.DatabaseURL(),
	)
}