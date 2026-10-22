package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

// AddItemToCartInput contains the trusted customer identity and requested item.
// CustomerID must come from the authenticated request context, not the body.
type AddItemToCartInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
	StoreID    uuid.UUID
	Quantity   int64
}

// AddItemToCart creates a cart on first use and persists the requested quantity.
// If the product is already in the cart, its quantity is overwritten.
type AddItemToCart struct {
	carts   ports.CartRepository
	logger  ports.Logger
	metrics ports.Metrics
}

func NewAddItemToCart(
	carts ports.CartRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *AddItemToCart {
	return &AddItemToCart{
		carts:   carts,
		logger:  logger,
		metrics: metrics,
	}
}

func (uc *AddItemToCart) Execute(
	ctx context.Context,
	input AddItemToCartInput,
) (*entities.Cart, error) {
	if input.CustomerID == uuid.Nil {
		return nil, domainerrors.ErrInvalidCustomerID
	}

	cart, err := uc.carts.GetByCustomerID(ctx, input.CustomerID)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrCartNotFound) {
			uc.recordFailure("cart.add_item.persistence_failed")
			return nil, fmt.Errorf("get customer cart: %w", err)
		}

		cart, err = entities.NewCart(input.CustomerID)
		if err != nil {
			return nil, err
		}
	}

	if err := cart.AddItem(input.ProductID, input.StoreID, input.Quantity); err != nil {
		return nil, err
	}

	if err := uc.carts.Save(ctx, cart); err != nil {
		uc.recordFailure("cart.add_item.persistence_failed")
		return nil, fmt.Errorf("save customer cart: %w", err)
	}

	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.item_added",
		Operation: "add_item_to_cart",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_item_mutations_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "add",
			"result":    "success",
		},
	})

	return cart, nil
}

func (uc *AddItemToCart) recordFailure(event string) {
	uc.logger.LogEvent(ports.LogEvent{
		Event:           event,
		Operation:       "add_item_to_cart",
		FailureCategory: "persistence",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_item_mutations_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "add",
			"result":    "failure",
		},
	})
}
