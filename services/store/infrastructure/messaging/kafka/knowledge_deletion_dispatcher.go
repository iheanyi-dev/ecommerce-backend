package kafka

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/events"
)

type KnowledgeDeletionDispatcher struct {
	publisher MessagePublisher
}

func NewKnowledgeDeletionDispatcher(publisher MessagePublisher) *KnowledgeDeletionDispatcher {
	return &KnowledgeDeletionDispatcher{publisher: publisher}
}

func (d *KnowledgeDeletionDispatcher) DispatchKnowledgeDeletion(
	ctx context.Context,
	request ports.KnowledgeDeletionRequested,
) error {
	if request.KnowledgeID == uuid.Nil {
		return errors.New("knowledge deletion dispatcher: knowledge ID is required")
	}
	if request.StoreID == uuid.Nil {
		return errors.New("knowledge deletion dispatcher: store ID is required")
	}
	return d.publisher.Publish(
		ctx,
		events.KnowledgeDeletionRequestedTopic,
		request.KnowledgeID.String(),
		events.KnowledgeDeletionRequested{
			KnowledgeID: request.KnowledgeID,
			StoreID:     request.StoreID,
		},
	)
}
