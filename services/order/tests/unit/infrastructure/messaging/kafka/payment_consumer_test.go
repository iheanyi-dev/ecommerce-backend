package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	kafkaconsumer "github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/messaging/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

type paymentEventHandlerMock struct {
	paymentCalls int
	payoutCalls  int
	orderID      string
	status       string
	paymentErr   error
	payoutErr    error
}

func (m *paymentEventHandlerMock) HandlePayment(_ context.Context, orderID, status string) error {
	m.paymentCalls++
	m.orderID = orderID
	m.status = status
	return m.paymentErr
}

func (m *paymentEventHandlerMock) HandlePayout(_ context.Context, orderID, status string) error {
	m.payoutCalls++
	m.orderID = orderID
	m.status = status
	return m.payoutErr
}

var _ ports.PaymentEventHandler = (*paymentEventHandlerMock)(nil)

func TestPaymentConsumerHandlePaymentRecord(t *testing.T) {
	const orderID = "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		payload    string
		handlerErr error
		wantCalls  int
		wantStatus string
		wantErr    bool
	}{
		{
			name: "handles successful payment",
			payload: `{
				"event_id":"event-1",
				"business_reference_type":"order",
				"business_reference":"` + orderID + `",
				"status":"succeeded"
			}`,
			wantCalls:  1,
			wantStatus: "succeeded",
		},
		{
			name: "handles failed payment",
			payload: `{
				"event_id":"event-2",
				"business_reference_type":"order",
				"business_reference":"` + orderID + `",
				"status":"failed"
			}`,
			wantCalls:  1,
			wantStatus: "failed",
		},
		{
			name: "ignores non-order business references",
			payload: `{
				"event_id":"event-3",
				"business_reference_type":"subscription",
				"business_reference":"` + orderID + `",
				"status":"succeeded"
			}`,
		},
		{
			name: "ignores non-terminal payment status",
			payload: `{
				"event_id":"event-4",
				"business_reference_type":"order",
				"business_reference":"` + orderID + `",
				"status":"processing"
			}`,
		},
		{
			name:    "discards malformed payload",
			payload: `{invalid`,
		},
		{
			name: "discards invalid order ID",
			payload: `{
				"business_reference_type":"order",
				"business_reference":"invalid",
				"status":"succeeded"
			}`,
		},
		{
			name: "returns handler error",
			payload: `{
				"business_reference_type":"order",
				"business_reference":"` + orderID + `",
				"status":"succeeded"
			}`,
			handlerErr: errors.New("database unavailable"),
			wantCalls:  1,
			wantStatus: "succeeded",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &paymentEventHandlerMock{paymentErr: tt.handlerErr}
			err := kafkaconsumer.HandlePaymentRecord(
				context.Background(),
				&kgo.Record{
					Value:   []byte(tt.payload),
					Headers: []kgo.RecordHeader{{Key: "event_type", Value: []byte("payment.status_changed")}},
				},
				handler,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandlePaymentRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
			if handler.paymentCalls != tt.wantCalls {
				t.Fatalf("payment calls = %d, want %d", handler.paymentCalls, tt.wantCalls)
			}
			if handler.status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", handler.status, tt.wantStatus)
			}
		})
	}
}

func TestPaymentConsumerHandlePayoutRecord(t *testing.T) {
	const orderID = "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		payload    string
		handlerErr error
		wantCalls  int
		wantStatus string
		wantErr    bool
	}{
		{
			name: "handles successful payout",
			payload: `{
				"event_id":"event-1",
				"payout_id":"payout-1",
				"order_id":"` + orderID + `",
				"status":"succeeded"
			}`,
			wantCalls:  1,
			wantStatus: "succeeded",
		},
		{
			name: "handles failed payout",
			payload: `{
				"event_id":"event-2",
				"payout_id":"payout-2",
				"order_id":"` + orderID + `",
				"status":"failed"
			}`,
			wantCalls:  1,
			wantStatus: "failed",
		},
		{
			name: "ignores non-terminal payout status",
			payload: `{
				"payout_id":"payout-3",
				"order_id":"` + orderID + `",
				"status":"processing"
			}`,
		},
		{
			name: "discards invalid order ID",
			payload: `{
				"payout_id":"payout-4",
				"order_id":"invalid",
				"status":"succeeded"
			}`,
		},
		{
			name: "returns handler error",
			payload: `{
				"payout_id":"payout-5",
				"order_id":"` + orderID + `",
				"status":"succeeded"
			}`,
			handlerErr: errors.New("database unavailable"),
			wantCalls:  1,
			wantStatus: "succeeded",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &paymentEventHandlerMock{payoutErr: tt.handlerErr}
			err := kafkaconsumer.HandlePaymentRecord(
				context.Background(),
				&kgo.Record{
					Value:   []byte(tt.payload),
					Headers: []kgo.RecordHeader{{Key: "event_type", Value: []byte("payout.status_changed")}},
				},
				handler,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandlePaymentRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
			if handler.payoutCalls != tt.wantCalls {
				t.Fatalf("payout calls = %d, want %d", handler.payoutCalls, tt.wantCalls)
			}
			if handler.status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", handler.status, tt.wantStatus)
			}
		})
	}
}

func TestPaymentConsumerDiscardsMissingOrUnsupportedEventType(t *testing.T) {
	const orderID = "123e4567-e89b-12d3-a456-426614174000"
	payload := []byte(`{"business_reference_type":"order","business_reference":"` + orderID + `","status":"succeeded"}`)

	tests := []struct {
		name    string
		headers []kgo.RecordHeader
	}{
		{name: "missing event type"},
		{
			name: "unsupported event type",
			headers: []kgo.RecordHeader{
				{Key: "event_type", Value: []byte("unknown.event")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &paymentEventHandlerMock{}
			err := kafkaconsumer.HandlePaymentRecord(
				context.Background(),
				&kgo.Record{Value: payload, Headers: tt.headers},
				handler,
			)
			if err != nil {
				t.Fatalf("HandlePaymentRecord() error = %v, want nil", err)
			}
			if handler.paymentCalls != 0 || handler.payoutCalls != 0 {
				t.Fatalf("unexpected handler calls: payment=%d payout=%d", handler.paymentCalls, handler.payoutCalls)
			}
		})
	}
}
