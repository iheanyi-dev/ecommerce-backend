package config_test

import (
	"net/url"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

func TestDatabaseURL_EscapesCredentials(t *testing.T) {
	cfg := &config.Config{
		DatabaseHost:     "localhost",
		DatabasePort:     "5432",
		DatabaseUser:     "promotion user",
		DatabasePassword: "p@ss/word",
		DatabaseName:     "promotion db",
		DatabaseSSLMode:  "disable",
	}

	parsed, err := url.Parse(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}

	username := parsed.User.Username()
	password, ok := parsed.User.Password()
	if username != cfg.DatabaseUser || !ok || password != cfg.DatabasePassword {
		t.Fatal("database URL did not preserve escaped credentials")
	}
	if parsed.Path != "/"+cfg.DatabaseName {
		t.Fatalf("database path = %q, want %q", parsed.Path, "/"+cfg.DatabaseName)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatal("database URL is missing sslmode")
	}
}
