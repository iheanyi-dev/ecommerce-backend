package kafka_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/messaging/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

type producerClientStub struct {
	record *kgo.Record
	err    error
	calls  int
}

func (c *producerClientStub) ProduceSync(
	_ context.Context,
	record *kgo.Record,
) error {
	c.calls++
	c.record = record
	return c.err
}

func (*producerClientStub) Close() {}

func TestProducerPublish(t *testing.T) {
	tests := []struct {
		name    string
		event   ports.PendingPromotionEvent
		client  *producerClientStub
		wantErr bool
	}{
		{
			name: "publishes event with key and event type header",
			event: ports.PendingPromotionEvent{
				EventID:     "event-123",
				PromotionID: "promotion-123",
				EventType:   "promotion.activated",
				Payload:     []byte(`{"status":"active"}`),
			},
			client: &producerClientStub{},
		},
		{
			name: "returns broker error",
			event: ports.PendingPromotionEvent{
				EventID:     "event-123",
				PromotionID: "promotion-123",
				EventType:   "promotion.activated",
				Payload:     []byte(`{"status":"active"}`),
			},
			client:  &producerClientStub{err: errors.New("broker unavailable")},
			wantErr: true,
		},
		{
			name: "rejects missing event ID",
			event: ports.PendingPromotionEvent{
				PromotionID: "promotion-123",
				EventType:   "promotion.activated",
				Payload:     []byte(`{"status":"active"}`),
			},
			client:  &producerClientStub{},
			wantErr: true,
		},
		{
			name: "rejects missing promotion ID",
			event: ports.PendingPromotionEvent{
				EventID:   "event-123",
				EventType: "promotion.activated",
				Payload:   []byte(`{"status":"active"}`),
			},
			client:  &producerClientStub{},
			wantErr: true,
		},
		{
			name: "rejects missing event type",
			event: ports.PendingPromotionEvent{
				EventID: "event-123",
				Payload: []byte(`{"status":"active"}`),
			},
			client:  &producerClientStub{},
			wantErr: true,
		},
		{
			name: "rejects empty payload",
			event: ports.PendingPromotionEvent{
				EventID:     "event-123",
				PromotionID: "promotion-123",
				EventType:   "promotion.activated",
			},
			client:  &producerClientStub{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := kafka.NewProducerWithClient(tt.client, "promotion.events")
			if err != nil {
				t.Fatalf("create producer: %v", err)
			}

			err = producer.Publish(context.Background(), tt.event)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				expectedCalls := 0
				if tt.event.EventID != "" &&
					tt.event.PromotionID != "" &&
					tt.event.EventType != "" &&
					len(tt.event.Payload) > 0 {
					expectedCalls = 1
				}

				if tt.client.calls != expectedCalls {
					t.Fatalf("expected %d broker calls, got %d",
						expectedCalls, tt.client.calls)
				}
				return
			}

			if tt.client.calls != 1 {
				t.Fatalf("expected one broker call, got %d", tt.client.calls)
			}

			record := tt.client.record
			if record.Topic != "promotion.events" {
				t.Fatalf("unexpected topic: %s", record.Topic)
			}
			if string(record.Key) != tt.event.PromotionID {
				t.Fatalf("unexpected key: %s", record.Key)
			}
			if string(record.Value) != string(tt.event.Payload) {
				t.Fatalf("unexpected payload: %s", record.Value)
			}
			if len(record.Headers) != 1 ||
				record.Headers[0].Key != "event_type" ||
				string(record.Headers[0].Value) != tt.event.EventType {
				t.Fatalf("unexpected headers: %+v", record.Headers)
			}
		})
	}
}

func TestNewProducerWithClientValidation(t *testing.T) {
	if _, err := kafka.NewProducerWithClient(nil, "promotion.events"); err == nil {
		t.Fatal("expected nil client to be rejected")
	}

	if _, err := kafka.NewProducerWithClient(&producerClientStub{}, ""); err == nil {
		t.Fatal("expected empty topic to be rejected")
	}
}
