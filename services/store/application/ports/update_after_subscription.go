package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// UpdateAfterSubscriptionService applies a trusted subscription update
// to its associated Store.
type UpdateAfterSubscriptionService interface {
	Execute(
		ctx context.Context,
		input dto.UpdateAfterSubscriptionInput,
	) (dto.StoreOutput, error)
}
