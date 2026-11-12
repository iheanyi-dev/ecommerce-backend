package kafka_test

import (
	"context"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/infrastructure/messaging/kafka"
)

type emailExecutorMock struct{}

func (emailExecutorMock) Execute(context.Context, entities.Email) error {
	return nil
}

func TestEmailEventConsumerHandleRecord(t *testing.T) {
	consumer := kafka.NewEmailEventConsumerForTest(
		"notification.email.events",
		emailExecutorMock{},
	)

	tests := []struct {
		name    string
		record  *kgo.Record
		wantErr bool
	}{
		{
			name:    "nil record",
			record:  nil,
			wantErr: true,
		},
		{
			name: "unexpected topic",
			record: &kgo.Record{
				Topic: "another.topic",
			},
			wantErr: true,
		},
		{
			name: "missing event type is discarded",
			record: &kgo.Record{
				Topic: "notification.email.events",
			},
		},
		{
			name: "unsupported event type is discarded",
			record: &kgo.Record{
				Topic:   "notification.email.events",
				Value:   []byte(`{}`),
				Headers: []kgo.RecordHeader{{Key: "event_type", Value: []byte("unknown.event")}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := consumer.HandleRecordForTest(context.Background(), tt.record)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleRecordForTest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
