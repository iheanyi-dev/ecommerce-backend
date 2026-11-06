package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ListBillingPlansService retrieves the Billing plan catalogue.
type ListBillingPlansService interface {
	Execute(ctx context.Context) ([]dto.BillingPlanOutput, error)
}

// GetBillingPlanService retrieves one Billing plan.
type GetBillingPlanService interface {
	Execute(ctx context.Context, input dto.GetBillingPlanInput) (dto.BillingPlanOutput, error)
}

// ListStoreSubscriptionsService retrieves subscriptions belonging to a Store.
type ListStoreSubscriptionsService interface {
	Execute(ctx context.Context, input dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error)
}

// GetStoreSubscriptionService retrieves one subscription belonging to a Store.
type GetStoreSubscriptionService interface {
	Execute(ctx context.Context, input dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error)
}

// BillingQueryIdentity is retained here as a small helper for query use cases
// that need to verify the authenticated Store owner.
func billingQueryIdentity(ctx context.Context) (AuthenticatedIdentity, bool) {
	identity, ok := AuthenticatedIdentityFromContext(ctx)
	return identity, ok && identity.UserID != uuid.Nil
}
