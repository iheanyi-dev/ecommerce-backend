package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// GetOrderInput identifies the order and the authenticated caller.
// A nonzero StoreID selects store scope; otherwise retrieval uses customer scope.
type GetOrderInput struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
	StoreID uuid.UUID
}

// OrderDetails contains an order and its persisted product lines.
type OrderDetails struct {
	Order    *entities.Order
	Products []*entities.OrderProduct
}

// GetOrder retrieves an order only when it matches the requested scope.
type GetOrder struct {
	orders   ports.OrderRepository
	products ports.OrderProductRepository
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewGetOrder constructs the order retrieval use case.
func NewGetOrder(
	orders ports.OrderRepository,
	products ports.OrderProductRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetOrder {
	return &GetOrder{
		orders: orders, products: products, logger: logger, metrics: metrics,
	}
}

// Execute retrieves one order and its product lines after enforcing scope.
func (uc *GetOrder) Execute(ctx context.Context, input GetOrderInput) (*OrderDetails, error) {
	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("get order: order ID is required")
	}
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("get order: authenticated user ID is required")
	}

	order, err := uc.orders.FindByID(ctx, input.OrderID)
	if err != nil {
		uc.log(ctx, "get_order", "load_order")
		return nil, fmt.Errorf("get order: load order: %w", err)
	}
	if order == nil || order.ID != input.OrderID {
		uc.log(ctx, "get_order", "load_order")
		return nil, fmt.Errorf("get order: order not found")
	}

	// Store scope takes precedence when supplied; otherwise use customer scope.
	if input.StoreID != uuid.Nil {
		if order.StoreID != input.StoreID {
			uc.log(ctx, "get_order", "scope_mismatch")
			return nil, fmt.Errorf("get order: order not found")
		}
	} else if order.CustomerID != input.UserID {
		uc.log(ctx, "get_order", "scope_mismatch")
		return nil, fmt.Errorf("get order: order not found")
	}

	lines, err := uc.products.FindByOrderIDs(ctx, []uuid.UUID{order.ID})
	if err != nil {
		uc.log(ctx, "get_order", "load_order_products")
		return nil, fmt.Errorf("get order: load order products: %w", err)
	}

	if err := validateOrderProductLines(lines, map[uuid.UUID]struct{}{
		order.ID: {},
	}); err != nil {
		uc.log(ctx, "get_order", "invalid_repository_result")
		return nil, fmt.Errorf("get order: invalid order products: %w", err)
	}

	uc.log(ctx, "get_order", "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name: "orders_retrieved", Value: 1,
		Labels: map[string]string{"operation": "get_order"},
	})

	return &OrderDetails{Order: order, Products: lines}, nil
}

// log records retrieval outcomes without changing the operation's result.
func (uc *GetOrder) log(ctx context.Context, operation, failureCategory string) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event: "order.retrieval", Operation: operation, FailureCategory: failureCategory,
	})
}
