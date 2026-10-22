package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

// SetCartItemQuantityInput identifies the authenticated customer, cart item,
// and absolute quantity requested.
type SetCartItemQuantityInput struct {
	CustomerID uuid.UUID
	ItemID     uuid.UUID
	Quantity   int64
}

// SetCartItemQuantity overwrites an existing cart item's quantity.
type SetCartItemQuantity struct {
	carts   ports.CartRepository
	logger  ports.Logger
	metrics ports.Metrics
}

func NewSetCartItemQuantity(
	carts ports.CartRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *SetCartItemQuantity {
	return &SetCartItemQuantity{carts: carts, logger: logger, metrics: metrics}
}

func (uc *SetCartItemQuantity) Execute(
	ctx context.Context,
	input SetCartItemQuantityInput,
) (*entities.Cart, error) {
	if input.CustomerID == uuid.Nil {
		return nil, domainerrors.ErrInvalidCustomerID
	}

	cart, err := uc.carts.GetByCustomerID(ctx, input.CustomerID)
	if err != nil {
		if err != domainerrors.ErrCartNotFound {
			uc.recordFailure("get")
			return nil, fmt.Errorf("get customer cart: %w", err)
		}
		return nil, err
	}

	if err := cart.SetItemQuantity(input.ItemID, input.Quantity); err != nil {
		return nil, err
	}

	if err := uc.carts.Save(ctx, cart); err != nil {
		uc.recordFailure("save")
		return nil, fmt.Errorf("save customer cart: %w", err)
	}

	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.item_quantity_set",
		Operation: "set_cart_item_quantity",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_item_mutations_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "set_quantity",
			"result":    "success",
		},
	})

	return cart, nil
}

func (uc *SetCartItemQuantity) recordFailure(stage string) {
	uc.logger.LogEvent(ports.LogEvent{
		Event:           "cart.set_quantity.persistence_failed",
		Operation:       "set_cart_item_quantity",
		FailureCategory: "persistence",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_item_mutations_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "set_quantity",
			"stage":     stage,
			"result":    "failure",
		},
	})
}
