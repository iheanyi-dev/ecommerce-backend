package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

type OrderClient interface {
	ListStoreOrders(ctx context.Context, storeID uuid.UUID, page, pageSize int) (*dto.ListStoreOrdersOutput, error)
	GetStoreOrder(ctx context.Context, storeID, orderID uuid.UUID) (*dto.OrderOutput, error)
	AddShipping(ctx context.Context, storeID, orderID uuid.UUID, shippingAmount int64) error
	MarkSent(ctx context.Context, storeID, orderID uuid.UUID) error
	CancelStoreOrder(ctx context.Context, storeID, orderID uuid.UUID) error
	InitiatePayout(ctx context.Context, storeID, orderID uuid.UUID, beneficiary PayoutBeneficiary) (*dto.InitiatePayoutOutput, error)
}
