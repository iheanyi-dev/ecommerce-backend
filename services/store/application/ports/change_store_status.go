package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ChangeStoreStatusService changes Store status for a trusted internal
// caller such as the Billing/Subscription service.
//
// Authorization of the internal caller belongs at the service boundary;
// the application use case performs the Store operation itself.
type ChangeStoreStatusService interface {
	Execute(ctx context.Context, input dto.ChangeStoreStatusInput) (dto.StoreOutput, error)
}
