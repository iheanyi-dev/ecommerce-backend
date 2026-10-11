package kafka

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/events"
)

// KnowledgeDispatcher publishes persisted Knowledge ingestion requests.
type KnowledgeDispatcher struct {
	publisher MessagePublisher
}

func NewKnowledgeDispatcher(publisher MessagePublisher) *KnowledgeDispatcher {
	return &KnowledgeDispatcher{publisher: publisher}
}

func (d *KnowledgeDispatcher) DispatchKnowledgeIngestion(
	ctx context.Context,
	request ports.KnowledgeIngestionRequested,
) error {
	if request.KnowledgeID == uuid.Nil {
		return errors.New("knowledge dispatcher: knowledge ID is required")
	}
	if request.StoreID == uuid.Nil {
		return errors.New("knowledge dispatcher: store ID is required")
	}
	if request.StorageReference == "" {
		return errors.New("knowledge dispatcher: storage reference is required")
	}

	return d.publisher.Publish(
		ctx,
		events.KnowledgeIngestionRequestedTopic,
		request.KnowledgeID.String(),
		events.KnowledgeIngestionRequested{
			KnowledgeID:      request.KnowledgeID,
			StoreID:          request.StoreID,
			StorageReference: request.StorageReference,
		},
	)
}
