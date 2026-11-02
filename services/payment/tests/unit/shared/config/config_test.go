package config_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

func setValidEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "test")
	t.Setenv("PAYMENT_HTTP_PORT", "18095")
	t.Setenv("PAYMENT_GRPC_PORT", "19095")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "payment")
	t.Setenv("DATABASE_PASSWORD", "test-password")
	t.Setenv("DATABASE_NAME", "payment_test")
	t.Setenv("DATABASE_SSL_MODE", "disable")
	t.Setenv("PAYMENT_TEST_DATABASE_URL", "")
	t.Setenv("PAYSTACK_SECRET_KEY", "sk_test_example")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.co")
	t.Setenv("SERVICE_AUTH_NAME", "payment")
	t.Setenv("SERVICE_AUTH_SECRET", "test-service-secret")
	t.Setenv("PAYMENT_BENEFICIARY_ENCRYPTION_KEY", strings.Repeat("a", 64))
}

func TestLoad(t *testing.T) {
	setValidEnvironment(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != "test" {
		t.Errorf("AppEnv = %q, want test", cfg.AppEnv)
	}

	if cfg.HTTPPort != "18095" {
		t.Errorf("HTTPPort = %q, want 18095", cfg.HTTPPort)
	}

	if cfg.GRPCPort != "19095" {
		t.Errorf("GRPCPort = %q, want 19095", cfg.GRPCPort)
	}

	if len(cfg.BeneficiaryEncryptionKey) != 32 {
		t.Errorf("BeneficiaryEncryptionKey length = %d, want 32", len(cfg.BeneficiaryEncryptionKey))
	}

	if cfg.PaystackSecretKey != "sk_test_example" {
		t.Errorf("PaystackSecretKey = %q, want configured test key", cfg.PaystackSecretKey)
	}

	if cfg.PaystackBaseURL != "https://api.paystack.co" {
		t.Errorf("PaystackBaseURL = %q, want production API URL", cfg.PaystackBaseURL)
	}
}

func TestLoadRequiresDatabaseUser(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DATABASE_USER", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_USER") {
		t.Fatalf("Load() error = %v, want DATABASE_USER validation error", err)
	}
}

func TestLoadRequiresDatabasePassword(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DATABASE_PASSWORD", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_PASSWORD") {
		t.Fatalf("Load() error = %v, want DATABASE_PASSWORD validation error", err)
	}
}

func TestLoadRequiresDatabaseName(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DATABASE_NAME", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_NAME") {
		t.Fatalf("Load() error = %v, want DATABASE_NAME validation error", err)
	}
}

func TestLoadRequiresPaystackSecret(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("PAYSTACK_SECRET_KEY", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "PAYSTACK_SECRET_KEY") {
		t.Fatalf("Load() error = %v, want PAYSTACK_SECRET_KEY validation error", err)
	}
}

func TestLoadRejectsInvalidPaystackBaseURL(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("PAYSTACK_BASE_URL", "file:///etc/passwd")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid Paystack URL error")
	}
}

func TestDatabaseURLEscapesCredentialsAndDatabaseName(t *testing.T) {
	cfg := &config.Config{
		DatabaseHost:     "localhost",
		DatabasePort:     "5432",
		DatabaseUser:     "payment user",
		DatabasePassword: "p@ss/word",
		DatabaseName:     "payment database",
		DatabaseSSLMode:  "disable",
	}

	parsed, err := url.Parse(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("url.Parse(DatabaseURL()) error = %v", err)
	}

	username := parsed.User.Username()
	password, _ := parsed.User.Password()

	if username != cfg.DatabaseUser {
		t.Errorf("database username = %q, want %q", username, cfg.DatabaseUser)
	}

	if password != cfg.DatabasePassword {
		t.Errorf("database password = %q, want configured password", password)
	}

	if parsed.Path != "/payment database" {
		t.Errorf("database path = %q, want /payment database", parsed.Path)
	}

	if parsed.Query().Get("sslmode") != "disable" {
		t.Errorf("sslmode = %q, want disable", parsed.Query().Get("sslmode"))
	}
}
