package kafka_test

import (
	"context"
	"testing"

	kafka "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/messaging/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestHandlePaymentRecordIgnoresInvalidEvents(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{
			name:  "malformed JSON",
			value: []byte(`{"event_id":`),
		},
		{
			name: "unrelated business reference",
			value: []byte(`{
				"business_reference_type":"order",
				"business_reference":"not-a-uuid",
				"status":"succeeded"
			}`),
		},
		{
			name: "non-terminal payment status",
			value: []byte(`{
				"business_reference_type":"subscription",
				"business_reference":"not-a-uuid",
				"status":"processing"
			}`),
		},
		{
			name: "missing business reference",
			value: []byte(`{
				"business_reference_type":"subscription",
				"status":"succeeded"
			}`),
		},
		{
			name: "invalid subscription UUID",
			value: []byte(`{
				"business_reference_type":"subscription",
				"business_reference":"not-a-uuid",
				"status":"succeeded"
			}`),
		},
		{
			name: "nil subscription UUID",
			value: []byte(`{
				"business_reference_type":"subscription",
				"business_reference":"00000000-0000-0000-0000-000000000000",
				"status":"failed"
			}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := kafka.HandlePaymentRecord(
				context.Background(),
				&kgo.Record{Value: test.value},
				nil,
			)
			if err != nil {
				t.Fatalf("HandlePaymentRecord() error = %v, want nil", err)
			}
		})
	}
}

func TestHandlePaymentRecordRejectsMissingDependencies(t *testing.T) {
	t.Run("nil record", func(t *testing.T) {
		err := kafka.HandlePaymentRecord(context.Background(), nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil record")
		}
	})

	t.Run("valid subscription event without update use case", func(t *testing.T) {
		record := &kgo.Record{Value: []byte(`{
			"business_reference_type":"subscription",
			"business_reference":"11111111-1111-4111-8111-111111111111",
			"status":"succeeded"
		}`)}
		err := kafka.HandlePaymentRecord(context.Background(), record, nil)
		if err == nil {
			t.Fatal("expected an error when the update use case is unavailable")
		}
	})
}
