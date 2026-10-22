package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

type RemoveCartItemInput struct {
	CustomerID uuid.UUID
	ItemID     uuid.UUID
}

type RemoveCartItem struct {
	carts   ports.CartRepository
	logger  ports.Logger
	metrics ports.Metrics
}

func NewRemoveCartItem(
	carts ports.CartRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *RemoveCartItem {
	return &RemoveCartItem{
		carts:   carts,
		logger:  logger,
		metrics: metrics,
	}
}

func (uc *RemoveCartItem) Execute(
	ctx context.Context,
	input RemoveCartItemInput,
) error {
	if input.CustomerID == uuid.Nil {
		return domainerrors.ErrInvalidCustomerID
	}
	if input.ItemID == uuid.Nil {
		return domainerrors.ErrCartItemNotFound
	}

	cart, err := uc.carts.GetByCustomerID(ctx, input.CustomerID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrCartNotFound) {
			return err
		}

		uc.recordFailure("remove_item", "repository")
		return fmt.Errorf("get cart for item removal: %w", err)
	}

	if err := cart.RemoveItem(input.ItemID); err != nil {
		return err
	}

	if err := uc.carts.Save(ctx, cart); err != nil {
		uc.recordFailure("remove_item", "repository")
		return fmt.Errorf("save cart after item removal: %w", err)
	}

	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.item_removed",
		Operation: "remove_item",
	})

	uc.metrics.Record(ports.Metric{
		Name:  "cart.item_removed",
		Value: 1,
		Labels: map[string]string{
			"operation": "remove_item",
			"result":    "success",
		},
	})

	return nil
}

func (uc *RemoveCartItem) recordFailure(operation, category string) {
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
