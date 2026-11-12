package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	AppPort  string
	GRPCPort string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string

	JWTSecret         string
	JWTIssuer         string
	JWTAccessTokenTTL time.Duration

	ActivationURL      string
	ActivationTokenTTL time.Duration

	ServiceAuthName   string
	ServiceAuthSecret string

	KafkaBrokers           []string
	NotificationEmailTopic string
}

func Load() (*Config, error) {
	// Resolve the service .env relative to this source file so loading
	// works regardless of the process working directory.
	_, sourceFile, _, ok := runtime.Caller(0)
	if ok {
		envPath := filepath.Join(filepath.Dir(sourceFile), "../../.env")
		_ = godotenv.Load(envPath)
	}

	jwtAccessTokenTTL, err := time.ParseDuration(
		os.Getenv("JWT_ACCESS_TOKEN_TTL"),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TOKEN_TTL: %w", err)
	}

	activationTokenTTL, err := time.ParseDuration(
		getEnv("ACTIVATION_TOKEN_TTL", "24h"),
	)
	if err != nil || activationTokenTTL <= 0 {
		return nil, fmt.Errorf("ACTIVATION_TOKEN_TTL must be a positive duration")
	}

	activationURL := getEnv(
		"ACTIVATION_URL",
		"http://localhost:3000/activate",
	)
	parsedActivationURL, err := url.Parse(activationURL)
	if err != nil || parsedActivationURL.Scheme == "" || parsedActivationURL.Host == "" {
		return nil, fmt.Errorf("ACTIVATION_URL must be an absolute URL")
	}

	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		AppPort:  getEnv("APP_PORT", "8080"),
		GRPCPort: getEnv("GRPC_PORT", "9090"),

		DatabaseHost:     getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:     os.Getenv("DATABASE_USER"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:     os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:  getEnv("DATABASE_SSL_MODE", "disable"),

		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTIssuer:          os.Getenv("JWT_ISSUER"),
		JWTAccessTokenTTL:  jwtAccessTokenTTL,
		ActivationURL:      activationURL,
		ActivationTokenTTL: activationTokenTTL,

		ServiceAuthName:   os.Getenv("SERVICE_AUTH_NAME"),
		ServiceAuthSecret: os.Getenv("SERVICE_AUTH_SECRET"),

		KafkaBrokers:           parseBrokers(os.Getenv("KAFKA_BROKERS")),
		NotificationEmailTopic: strings.TrimSpace(os.Getenv("NOTIFICATION_EMAIL_EVENTS_TOPIC")),
	}

	if cfg.DatabaseUser == "" {
		return nil, fmt.Errorf("DATABASE_USER is required")
	}
	if cfg.DatabasePassword == "" {
		return nil, fmt.Errorf("DATABASE_PASSWORD is required")
	}
	if cfg.DatabaseName == "" {
		return nil, fmt.Errorf("DATABASE_NAME is required")
	}
	if cfg.ServiceAuthName == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_NAME is required")
	}
	if cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("SERVICE_AUTH_SECRET is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("at least one KAFKA_BROKERS entry is required")
	}
	if cfg.NotificationEmailTopic == "" {
		return nil, fmt.Errorf("NOTIFICATION_EMAIL_EVENTS_TOPIC is required")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DatabaseUser, c.DatabasePassword),
		Host:   fmt.Sprintf("%s:%s", c.DatabaseHost, c.DatabasePort),
		Path:   c.DatabaseName,
	}

	query := u.Query()
	query.Set("sslmode", c.DatabaseSSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseBrokers(value string) []string {
	var brokers []string
	for _, broker := range strings.Split(value, ",") {
		if broker = strings.TrimSpace(broker); broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}
