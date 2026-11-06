package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// GetSubByIDInput identifies the subscription to retrieve.
type GetSubByIDInput struct {
	SubscriptionID uuid.UUID
}

// GetSubByIDOutput contains the requested subscription.
type GetSubByIDOutput struct {
	Subscription entities.Subscription
}

// GetSubByID retrieves a subscription by its unique identifier.
type GetSubByID struct {
	repository ports.SubscriptionRepository
}

// NewGetSubByID creates a subscription lookup query.
func NewGetSubByID(repository ports.SubscriptionRepository) *GetSubByID {
	return &GetSubByID{repository: repository}
}

// Execute retrieves the subscription identified by the input UUID.
func (useCase *GetSubByID) Execute(
	ctx context.Context,
	input GetSubByIDInput,
) (GetSubByIDOutput, error) {
	if useCase == nil || useCase.repository == nil {
		return GetSubByIDOutput{}, application.ErrSubscriptionRepositoryUnavailable
	}
	if input.SubscriptionID == uuid.Nil {
		return GetSubByIDOutput{}, domain.ErrInvalidSubscription
	}

	subscription, err := useCase.repository.GetByID(ctx, input.SubscriptionID)
	if err != nil {
		if errors.Is(err, application.ErrSubscriptionNotFound) {
			return GetSubByIDOutput{}, application.ErrSubscriptionNotFound
		}
		return GetSubByIDOutput{}, fmt.Errorf("get subscription by ID: %w", err)
	}

	if subscription == nil || subscription.ID != input.SubscriptionID {
		return GetSubByIDOutput{}, application.ErrSubscriptionNotFound
	}

	return GetSubByIDOutput{Subscription: *subscription}, nil
}
