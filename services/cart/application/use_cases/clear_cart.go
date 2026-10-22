package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

type ClearCart struct {
	carts   ports.CartRepository
	logger  ports.Logger
	metrics ports.Metrics
}

func NewClearCart(
	carts ports.CartRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *ClearCart {
	return &ClearCart{
		carts:   carts,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute empties an existing cart without deleting the cart itself.
// Saving even an already-empty cart allows the repository to refresh its TTL.
func (uc *ClearCart) Execute(ctx context.Context, customerID uuid.UUID) error {
	if customerID == uuid.Nil {
		return domainerrors.ErrInvalidCustomerID
	}

	cart, err := uc.carts.GetByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrCartNotFound) {
			return err
		}

		uc.recordFailure("clear_cart", "repository")
		return fmt.Errorf("get cart to clear: %w", err)
	}

	cart.Clear()

	if err := uc.carts.Save(ctx, cart); err != nil {
		uc.recordFailure("clear_cart", "repository")
		return fmt.Errorf("save cleared cart: %w", err)
	}

	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.cleared",
		Operation: "clear_cart",
	})

	uc.metrics.Record(ports.Metric{
		Name:  "cart.cleared",
		Value: 1,
		Labels: map[string]string{
			"operation": "clear_cart",
			"result":    "success",
		},
	})

	return nil
}

func (uc *ClearCart) recordFailure(operation, category string) {
	uc.logger.LogEvent(ports.LogEvent{
		Event:           "cart.operation_failed",
		Operation:       operation,
		FailureCategory: category,
	})

	uc.metrics.Record(ports.Metric{
		Name:  "cart.operation_failed",
		Value: 1,
		Labels: map[string]string{
			"operation": operation,
			"category":  category,
			"result":    "failure",
		},
	})
}
