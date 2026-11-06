package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// SubscribeService initiates a Store subscription checkout.
type SubscribeService interface {
	Execute(ctx context.Context, input dto.SubscribeInput) (dto.SubscribeOutput, error)
}
