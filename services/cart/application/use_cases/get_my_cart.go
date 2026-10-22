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

// GetMyCart retrieves the authenticated customer's existing cart.
type GetMyCart struct {
	carts   ports.CartRepository
	logger  ports.Logger
	metrics ports.Metrics
}

func NewGetMyCart(
	carts ports.CartRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetMyCart {
	return &GetMyCart{carts: carts, logger: logger, metrics: metrics}
}

func (uc *GetMyCart) Execute(
	ctx context.Context,
	customerID uuid.UUID,
) (*entities.Cart, error) {
	if customerID == uuid.Nil {
		return nil, domainerrors.ErrInvalidCustomerID
	}

	cart, err := uc.carts.GetByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrCartNotFound) {
			return nil, err
		}

		uc.recordFailure()
		return nil, fmt.Errorf("get customer cart: %w", err)
	}

	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.retrieved",
		Operation: "get_my_cart",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_reads_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "get",
			"result":    "success",
		},
	})

	return cart, nil
}

func (uc *GetMyCart) recordFailure() {
	uc.logger.LogEvent(ports.LogEvent{
		Event:           "cart.get.persistence_failed",
		Operation:       "get_my_cart",
		FailureCategory: "persistence",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_reads_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "get",
			"result":    "failure",
		},
	})
}
