package kafka_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/events"
	kafkainfra "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/kafka"
)

type fakePromotionHandler struct {
	addCalls    int
	updateCalls int
	removeCalls int

	addInput    use_cases.AddProductPromotionInput
	updateInput use_cases.UpdateProductPromotionInput
	removeInput use_cases.RemoveProductPromotionInput

	err error
}

func (h *fakePromotionHandler) Add(
	_ context.Context,
	input use_cases.AddProductPromotionInput,
) error {
	h.addCalls++
	h.addInput = input
	return h.err
}

func (h *fakePromotionHandler) Update(
	_ context.Context,
	input use_cases.UpdateProductPromotionInput,
) error {
	h.updateCalls++
	h.updateInput = input
	return h.err
}

func (h *fakePromotionHandler) Remove(
	_ context.Context,
	input use_cases.RemoveProductPromotionInput,
) error {
	h.removeCalls++
	h.removeInput = input
	return h.err
}

func promotionRecord(t *testing.T, eventType string) *kgo.Record {
	t.Helper()

	productID := "33333333-3333-4333-8333-333333333333"
	payload, err := json.Marshal(events.PromotionEvent{
		PromoID:   "11111111-1111-4111-8111-111111111111",
		StoreID:   "22222222-2222-4222-8222-222222222222",
		ProductID: &productID,
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	return &kgo.Record{
		Topic: "promotion.events",
		Value: payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(eventType)},
		},
	}
}

func TestHandlePromotionRecordRoutesEvents(t *testing.T) {
	tests := []struct {
		name       string
		eventType  string
		wantAdd    int
		wantUpdate int
		wantRemove int
	}{
		{name: "created adds snapshot", eventType: events.PromotionCreated, wantAdd: 1},
		{name: "activated updates snapshot", eventType: events.PromotionActivated, wantUpdate: 1},
		{name: "renewed recreates snapshot", eventType: events.PromotionRenewed, wantAdd: 1},
		{name: "cancelled removes snapshot", eventType: events.PromotionCancelled, wantRemove: 1},
		{name: "expired removes snapshot", eventType: events.PromotionExpired, wantRemove: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &fakePromotionHandler{}
			record := promotionRecord(t, tt.eventType)

			if err := kafkainfra.HandlePromotionRecord(
				context.Background(),
				record,
				handler,
			); err != nil {
				t.Fatalf("HandlePromotionRecord() error = %v", err)
			}

			if handler.addCalls != tt.wantAdd {
				t.Errorf("add calls = %d, want %d", handler.addCalls, tt.wantAdd)
			}
			if handler.updateCalls != tt.wantUpdate {
				t.Errorf("update calls = %d, want %d", handler.updateCalls, tt.wantUpdate)
			}
			if handler.removeCalls != tt.wantRemove {
				t.Errorf("remove calls = %d, want %d", handler.removeCalls, tt.wantRemove)
			}
		})
	}
}

func TestHandlePromotionRecordPassesSnapshotFields(t *testing.T) {
	handler := &fakePromotionHandler{}
	record := promotionRecord(t, events.PromotionCreated)

	if err := kafkainfra.HandlePromotionRecord(
		context.Background(),
		record,
		handler,
	); err != nil {
		t.Fatalf("HandlePromotionRecord() error = %v", err)
	}

	if handler.addInput.StoreID != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("store ID = %q", handler.addInput.StoreID)
	}
	if handler.addInput.PromotionID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("promotion ID = %q", handler.addInput.PromotionID)
	}
	if handler.addInput.ProductID == nil ||
		*handler.addInput.ProductID != "33333333-3333-4333-8333-333333333333" {
		t.Errorf("product ID = %v", handler.addInput.ProductID)
	}
	if handler.addInput.Status != "active" {
		t.Errorf("status = %q", handler.addInput.Status)
	}
}

func TestHandlePromotionRecordIgnoresUnsupportedEvent(t *testing.T) {
	handler := &fakePromotionHandler{}

	err := kafkainfra.HandlePromotionRecord(
		context.Background(),
		promotionRecord(t, "promotion.something_new"),
		handler,
	)
	if err != nil {
		t.Fatalf("HandlePromotionRecord() error = %v", err)
	}

	if handler.addCalls+handler.updateCalls+handler.removeCalls != 0 {
		t.Fatal("unsupported event was dispatched")
	}
}

func TestHandlePromotionRecordRejectsMalformedPayload(t *testing.T) {
	handler := &fakePromotionHandler{}
	record := promotionRecord(t, events.PromotionCreated)
	record.Value = []byte("{invalid")

	if err := kafkainfra.HandlePromotionRecord(
		context.Background(),
		record,
		handler,
	); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestHandlePromotionRecordReturnsUseCaseError(t *testing.T) {
	handler := &fakePromotionHandler{err: context.DeadlineExceeded}

	err := kafkainfra.HandlePromotionRecord(
		context.Background(),
		promotionRecord(t, events.PromotionCreated),
		handler,
	)
	if err == nil {
		t.Fatal("expected handler error")
	}
}
