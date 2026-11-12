package notification

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	KafkaBrokers              []string
	NotificationEmailTopic    string
	NotificationEmailConsumer string
}

func LoadConfig() (*Config, error) {
	_, file, _, ok := runtime.Caller(0)
	if ok {
		_ = godotenv.Load(filepath.Join(filepath.Dir(file), ".env"))
	}

	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid SMTP_PORT")
	}

	brokers := splitCSV(os.Getenv("KAFKA_BROKERS"))

	config := &Config{
		SMTPHost:                  strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                  port,
		SMTPUsername:              os.Getenv("SMTP_USERNAME"),
		SMTPPassword:              os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:                  strings.TrimSpace(os.Getenv("SMTP_FROM")),
		KafkaBrokers:              brokers,
		NotificationEmailTopic:    strings.TrimSpace(os.Getenv("NOTIFICATION_EMAIL_EVENTS_TOPIC")),
		NotificationEmailConsumer: strings.TrimSpace(os.Getenv("NOTIFICATION_EMAIL_CONSUMER_GROUP")),
	}

	if config.SMTPHost == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	if config.SMTPFrom == "" {
		return nil, fmt.Errorf("SMTP_FROM is required")
	}
	if (config.SMTPUsername == "") != (config.SMTPPassword == "") {
		return nil, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must both be provided")
	}
	if len(config.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS is required")
	}
	if config.NotificationEmailTopic == "" {
		return nil, fmt.Errorf("NOTIFICATION_EMAIL_EVENTS_TOPIC is required")
	}
	if config.NotificationEmailConsumer == "" {
		return nil, fmt.Errorf("NOTIFICATION_EMAIL_CONSUMER_GROUP is required")
	}

	return config, nil
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
