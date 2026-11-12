package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

const activationRequestedEvent = "identity.activation_requested"

type ProducerClient interface {
	ProduceSync(context.Context, ...*kgo.Record) kgo.ProduceResults
	Close()
}

type ActivationEmailSender struct {
	client ProducerClient
	topic  string
}

type activationRequestedPayload struct {
	Recipient     string `json:"recipient"`
	ActivationURL string `json:"activation_url"`
}

func NewActivationEmailSender(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
) (*ActivationEmailSender, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Identity configuration is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("at least one Kafka broker is required")
	}
	if cfg.NotificationEmailTopic == "" {
		return nil, fmt.Errorf("notification email topic is required")
	}

	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.KafkaBrokers...))
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}

	sender := &ActivationEmailSender{
		client: client,
		topic:  cfg.NotificationEmailTopic,
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			sender.Close()
			return nil
		},
	})

	return sender, nil
}

func NewActivationEmailSenderWithClient(
	client ProducerClient,
	topic string,
) (*ActivationEmailSender, error) {
	if client == nil {
		return nil, fmt.Errorf("Kafka producer client is required")
	}
	if strings.TrimSpace(topic) == "" {
		return nil, fmt.Errorf("notification email topic is required")
	}

	return &ActivationEmailSender{client: client, topic: topic}, nil
}

func (s *ActivationEmailSender) SendActivationEmail(
	ctx context.Context,
	recipient user.Email,
	activationURL string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	email := strings.TrimSpace(recipient.String())
	activationURL = strings.TrimSpace(activationURL)
	if email == "" || activationURL == "" {
		return fmt.Errorf("activation email recipient and URL are required")
	}

	payload, err := json.Marshal(activationRequestedPayload{
		Recipient:     email,
		ActivationURL: activationURL,
	})
	if err != nil {
		return fmt.Errorf("marshal activation email event: %w", err)
	}

	record := &kgo.Record{
		Topic: s.topic,
		Key:   []byte(email),
		Value: payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(activationRequestedEvent)},
		},
	}

	if err := s.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish activation email event: %w", err)
	}

	return nil
}

func (s *ActivationEmailSender) Close() {
	s.client.Close()
}

var _ ports.ActivationEmailSender = (*ActivationEmailSender)(nil)
