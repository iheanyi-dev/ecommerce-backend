package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config contains configuration required by the Storage service.
type Config struct {
	AppEnv       string
	AppPort      string
	GRPCPort     string
	KafkaBrokers string

	// StorageRoot is the filesystem root under which all object references
	// are stored.
	//
	// Calling services own the object-reference conventions, for example:
	//
	//	stores/{storeID}.png
	//	products/{storeID}/{productID}.webp
	//	knowledge/{storeID}/{filename}.pdf
	StorageRoot string

	// Service authentication configuration is used later by the Storage
	// presentation layer to protect internal-only operations.
	ServiceAuthName   string
	ServiceAuthSecret string
}

// Load loads Storage configuration from the environment.
func Load() (*Config, error) {
	_ = godotenv.Load("services/storage/.env")

	cfg := &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		AppPort:           getEnv("APP_PORT", "8083"),
		GRPCPort:          getEnv("GRPC_PORT", "9082"),
		KafkaBrokers:      getEnv("KAFKA_BROKERS", "localhost:19092"),
		StorageRoot:       getEnv("STORAGE_ROOT", "./storage-data"),
		ServiceAuthName:   os.Getenv("SERVICE_AUTH_NAME"),
		ServiceAuthSecret: os.Getenv("SERVICE_AUTH_SECRET"),
	}

	if cfg.StorageRoot == "" {
		return nil, fmt.Errorf("STORAGE_ROOT is required")
	}

	if cfg.ServiceAuthName == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_NAME is required")
	}

	if cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return fallback
	}

	return result
}
