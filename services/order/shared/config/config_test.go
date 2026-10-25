package config_test

import (
	"net/url"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

// setValidDatabaseEnvironment supplies the required values so tests do not
// depend on credentials from the developer's local .env file.
func setValidDatabaseEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("DATABASE_USER", "order-user")
	t.Setenv("DATABASE_PASSWORD", "order-password")
	t.Setenv("DATABASE_NAME", "order-db")
}

func TestLoadRequiresDatabaseCredentials(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "database user", key: "DATABASE_USER"},
		{name: "database password", key: "DATABASE_PASSWORD"},
		{name: "database name", key: "DATABASE_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidDatabaseEnvironment(t)
			t.Setenv(tt.key, "")

			cfg, err := config.Load()
			if err == nil {
				t.Fatal("expected missing database credential to return an error")
			}

			if cfg != nil {
				t.Fatal("expected nil config when required credentials are missing")
			}
		})
	}
}

func TestLoadUsesDatabaseDefaults(t *testing.T) {
	setValidDatabaseEnvironment(t)
	t.Setenv("DATABASE_HOST", "")
	t.Setenv("DATABASE_PORT", "")
	t.Setenv("DATABASE_SSL_MODE", "")
	t.Setenv("APP_ENV", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.DatabaseHost != "localhost" {
		t.Errorf("DatabaseHost = %q, want %q", cfg.DatabaseHost, "localhost")
	}

	if cfg.DatabasePort != "5432" {
		t.Errorf("DatabasePort = %q, want %q", cfg.DatabasePort, "5432")
	}

	if cfg.DatabaseSSLMode != "disable" {
		t.Errorf("DatabaseSSLMode = %q, want %q", cfg.DatabaseSSLMode, "disable")
	}

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "development")
	}
}

func TestDatabaseURLEscapesCredentials(t *testing.T) {
	cfg := &config.Config{
		DatabaseHost:     "localhost",
		DatabasePort:     "5432",
		DatabaseUser:     "order user",
		DatabasePassword: "p@ss:/word?#%",
		DatabaseName:     "order",
		DatabaseSSLMode:  "disable",
	}

	parsed, err := url.Parse(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("DatabaseURL() produced an invalid URL: %v", err)
	}

	if parsed.Scheme != "postgres" {
		t.Errorf("URL scheme = %q, want %q", parsed.Scheme, "postgres")
	}

	if parsed.Host != "localhost:5432" {
		t.Errorf("URL host = %q, want %q", parsed.Host, "localhost:5432")
	}

	if parsed.Path != "/order" {
		t.Errorf("URL path = %q, want %q", parsed.Path, "/order")
	}

	username := parsed.User.Username()
	if username != cfg.DatabaseUser {
		t.Errorf("URL username = %q, want original username", username)
	}

	password, ok := parsed.User.Password()
	if !ok {
		t.Fatal("expected URL to contain a password")
	}

	if password != cfg.DatabasePassword {
		t.Errorf("URL password did not round-trip correctly")
	}

	if got := parsed.Query().Get("sslmode"); got != "disable" {
		t.Errorf("sslmode = %q, want %q", got, "disable")
	}
}
