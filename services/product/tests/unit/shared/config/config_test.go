package config_test

import (
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

func setRequiredEnvironment(t *testing.T) {
	t.Helper()

	values := map[string]string{
		"APP_ENV":                  "test",
		"GRPC_PORT":                "9091",
		"GRPC_SERVICE_AUTH_NAME":   "store",
		"GRPC_SERVICE_AUTH_SECRET": "test-grpc-secret",
		"DATABASE_HOST":            "localhost",
		"DATABASE_PORT":            "5432",
		"DATABASE_USER":            "product",
		"DATABASE_PASSWORD":        "test-password",
		"DATABASE_NAME":            "product_test",
		"DATABASE_SSL_MODE":        "disable",
		"STORAGE_SERVICE_ADDR":     "localhost:9082",
		"STORE_SERVICE_ADDR":       "localhost:9091",
		"SERVICE_AUTH_NAME":        "product",
		"SERVICE_AUTH_SECRET":      "test-service-secret",
	}

	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoadRequiresStoreServiceAddr(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("STORE_SERVICE_ADDR", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error when STORE_SERVICE_ADDR is missing")
	}

	if !strings.Contains(err.Error(), "STORE_SERVICE_ADDR") {
		t.Fatalf("expected error to mention STORE_SERVICE_ADDR, got: %v", err)
	}
}

func TestLoadReadsStoreServiceAddr(t *testing.T) {
	setRequiredEnvironment(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.StoreServiceAddr != "localhost:9091" {
		t.Fatalf(
			"StoreServiceAddr = %q, want %q",
			cfg.StoreServiceAddr,
			"localhost:9091",
		)
	}
}
