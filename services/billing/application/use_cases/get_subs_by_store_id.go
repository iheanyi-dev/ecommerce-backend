package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// GetSubsByStoreIDInput identifies the store whose subscription history
// should be retrieved.
type GetSubsByStoreIDInput struct {
	StoreID uuid.UUID
}

// GetSubsByStoreIDOutput contains all subscriptions belonging to a store.
type GetSubsByStoreIDOutput struct {
	Subscriptions []entities.Subscription
}

// GetSubsByStoreID retrieves the subscription history for one store.
type GetSubsByStoreID struct {
	repository ports.SubscriptionRepository
}

// NewGetSubsByStoreID creates a store subscription history query.
func NewGetSubsByStoreID(repository ports.SubscriptionRepository) *GetSubsByStoreID {
	return &GetSubsByStoreID{repository: repository}
}

// Execute retrieves all subscriptions belonging to the requested store.
func (useCase *GetSubsByStoreID) Execute(
	ctx context.Context,
	input GetSubsByStoreIDInput,
) (GetSubsByStoreIDOutput, error) {
	if useCase == nil || useCase.repository == nil {
		return GetSubsByStoreIDOutput{}, application.ErrSubscriptionRepositoryUnavailable
	}
	if input.StoreID == uuid.Nil {
		return GetSubsByStoreIDOutput{}, domain.ErrInvalidSubscription
	}

	subscriptions, err := useCase.repository.GetByStoreID(ctx, input.StoreID)
	if err != nil {
		return GetSubsByStoreIDOutput{}, fmt.Errorf("get subscriptions by store ID: %w", err)
	}

	if subscriptions == nil {
		subscriptions = make([]entities.Subscription, 0)
	}

	return GetSubsByStoreIDOutput{Subscriptions: subscriptions}, nil
}
