package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/events"
)

type recordingPublisher struct {
	topic string
	key   string
	value any
	err   error
	calls int
}

func (p *recordingPublisher) Publish(_ context.Context, topic, key string, value any) error {
	p.topic, p.key, p.value = topic, key, value
	p.calls++
	return p.err
}

func TestDispatchProductEmbeddingPublishesContract(t *testing.T) {
	productID := uuid.New()
	storeID := uuid.New()
	publisher := &recordingPublisher{}
	dispatcher := NewEmbeddingDispatcher(publisher)

	err := dispatcher.DispatchProductEmbedding(context.Background(), ports.ProductEmbedding{
		ProductID: productID.String(), StoreID: storeID.String(),
		Name: "Test product", Description: "Test description",
	})
	if err != nil {
		t.Fatalf("DispatchProductEmbedding() error = %v", err)
	}
	if publisher.topic != events.ProductEmbeddingRequestedTopic {
		t.Fatalf("topic = %q", publisher.topic)
	}
	if publisher.key != productID.String() {
		t.Fatalf("key = %q", publisher.key)
	}
	event, ok := publisher.value.(events.ProductEmbeddingRequested)
	if !ok {
		t.Fatalf("payload type = %T", publisher.value)
	}
	if event.ProductID != productID || event.StoreID != storeID ||
		event.Name != "Test product" || event.Description != "Test description" {
		t.Fatalf("unexpected event payload: %+v", event)
	}
}

func TestDispatchProductEmbeddingRejectsInvalidIDs(t *testing.T) {
	for _, test := range []struct {
		name    string
		product ports.ProductEmbedding
	}{
		{name: "invalid product ID", product: ports.ProductEmbedding{
			ProductID: "invalid", StoreID: uuid.NewString(),
		}},
		{name: "invalid store ID", product: ports.ProductEmbedding{
			ProductID: uuid.NewString(), StoreID: "invalid",
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			publisher := &recordingPublisher{}
			err := NewEmbeddingDispatcher(publisher).DispatchProductEmbedding(
				context.Background(), test.product,
			)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if publisher.calls != 0 {
				t.Fatal("publisher called for invalid IDs")
			}
		})
	}
}

func TestDispatchProductEmbeddingDeletionPublishesContract(t *testing.T) {
	productID, storeID := uuid.New(), uuid.New()
	publisher := &recordingPublisher{}
	err := NewEmbeddingDispatcher(publisher).DispatchProductEmbeddingDeletion(
		context.Background(), productID.String(), storeID.String(),
	)
	if err != nil {
		t.Fatalf("DispatchProductEmbeddingDeletion() error = %v", err)
	}
	if publisher.topic != events.ProductEmbeddingDeletionRequestedTopic {
		t.Fatalf("topic = %q", publisher.topic)
	}
	if publisher.key != productID.String() {
		t.Fatalf("key = %q", publisher.key)
	}
	event, ok := publisher.value.(events.ProductEmbeddingDeletionRequested)
	if !ok {
		t.Fatalf("payload type = %T", publisher.value)
	}
	if event.ProductID != productID || event.StoreID != storeID {
		t.Fatalf("unexpected event payload: %+v", event)
	}
}

func TestDispatchProductEmbeddingPropagatesPublishFailure(t *testing.T) {
	expected := errors.New("Kafka unavailable")
	publisher := &recordingPublisher{err: expected}
	err := NewEmbeddingDispatcher(publisher).DispatchProductEmbedding(
		context.Background(), ports.ProductEmbedding{
			ProductID: uuid.NewString(), StoreID: uuid.NewString(),
			Name: "Product", Description: "Description",
		},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want wrapped %v", err, expected)
	}
}
