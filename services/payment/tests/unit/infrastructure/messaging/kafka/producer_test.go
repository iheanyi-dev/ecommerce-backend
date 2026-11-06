package kafka_test

import (
	"context"
	"errors"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	kafka "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/messaging/kafka"
)

type mockProducerClient struct {
	record *kgo.Record
	err    error
}

func (m *mockProducerClient) ProduceSync(_ context.Context, records ...*kgo.Record) kgo.ProduceResults {
	if len(records) > 0 {
		m.record = records[0]
	}
	return kgo.ProduceResults{{Err: m.err}}
}

func (m *mockProducerClient) Close() {}

func TestProducerPublishIncludesEventTypeHeader(t *testing.T) {
	client := &mockProducerClient{}
	producer, err := kafka.NewProducerWithClient(client, "payment.events")
	if err != nil {
		t.Fatal(err)
	}

	event := ports.PaymentOutboxEvent{
		ID:        "event-1",
		EventType: "payment.status_changed",
		Payload:   []byte(`{"status":"succeeded"}`),
	}

	if err := producer.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if client.record == nil {
		t.Fatal("expected Kafka record to be produced")
	}
	if client.record.Topic != "payment.events" {
		t.Errorf("record topic = %q, want %q", client.record.Topic, "payment.events")
	}
	if string(client.record.Key) != event.ID {
		t.Errorf("record key = %q, want %q", client.record.Key, event.ID)
	}
	if string(client.record.Value) != string(event.Payload) {
		t.Errorf("record payload = %q, want %q", client.record.Value, event.Payload)
	}

	var eventType string
	for _, header := range client.record.Headers {
		if header.Key == "event_type" {
			eventType = string(header.Value)
			break
		}
	}
	if eventType != event.EventType {
		t.Errorf("event_type header = %q, want %q", eventType, event.EventType)
	}
}

func TestProducerPublishReturnsKafkaError(t *testing.T) {
	publishErr := errors.New("Kafka unavailable")
	producer, err := kafka.NewProducerWithClient(&mockProducerClient{err: publishErr}, "payment.events")
	if err != nil {
		t.Fatal(err)
	}

	err = producer.Publish(context.Background(), ports.PaymentOutboxEvent{
		ID:        "event-1",
		EventType: "payment.status_changed",
		Payload:   []byte(`{"status":"succeeded"}`),
	})
	if !errors.Is(err, publishErr) {
		t.Fatalf("Publish() error = %v, want %v", err, publishErr)
	}
}

func TestProducerPublishRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		event ports.PaymentOutboxEvent
	}{
		{
			name: "missing event ID",
			event: ports.PaymentOutboxEvent{
				EventType: "payment.status_changed",
				Payload:   []byte(`{"status":"succeeded"}`),
			},
		},
		{
			name: "missing event type",
			event: ports.PaymentOutboxEvent{
				ID:      "event-1",
				Payload: []byte(`{"status":"succeeded"}`),
			},
		},
		{
			name: "missing payload",
			event: ports.PaymentOutboxEvent{
				ID:        "event-1",
				EventType: "payment.status_changed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := kafka.NewProducerWithClient(&mockProducerClient{}, "payment.events")
			if err != nil {
				t.Fatal(err)
			}
			if err := producer.Publish(context.Background(), tt.event); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
