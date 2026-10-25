package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

type ListStoreOrdersService interface {
	Execute(context.Context, int, int) (*dto.ListStoreOrdersOutput, error)
}

type GetStoreOrderService interface {
	Execute(context.Context, uuid.UUID) (*dto.OrderOutput, error)
}

type AddShippingService interface {
	Execute(context.Context, uuid.UUID, int64) error
}

type MarkOrderSentService interface {
	Execute(context.Context, uuid.UUID) error
}

type CancelStoreOrderService interface {
	Execute(context.Context, uuid.UUID) error
}
