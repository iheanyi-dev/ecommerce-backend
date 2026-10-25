package use_cases

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

const (
	// DefaultOrderPageSize is used when callers omit a page size.
	DefaultOrderPageSize = 20
	// MaxOrderPageSize prevents unbounded or excessively large listing requests.
	MaxOrderPageSize = 100
)

// ListOrdersInput defines the scope and pagination for an order listing.
type ListOrdersInput struct {
	UserID   uuid.UUID
	StoreID  uuid.UUID
	Page     int
	PageSize int
}

// OrderPage contains a bounded result set and pagination metadata.
type OrderPage struct {
	Items      []*OrderDetails
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int64
}

// ListOrders returns a paginated list of orders for a customer or store.
type ListOrders struct {
	orders   ports.OrderRepository
	products ports.OrderProductRepository
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewListOrders constructs the paginated order listing use case.
func NewListOrders(
	orders ports.OrderRepository,
	products ports.OrderProductRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *ListOrders {
	return &ListOrders{
		orders: orders, products: products, logger: logger, metrics: metrics,
	}
}

// Execute lists orders in store scope when StoreID is provided, otherwise in
// authenticated-customer scope. Product lines are loaded in one batch.
func (uc *ListOrders) Execute(ctx context.Context, input ListOrdersInput) (*OrderPage, error) {
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("list orders: authenticated user ID is required")
	}
	if input.Page < 1 {
		return nil, fmt.Errorf("list orders: page must be at least 1")
	}

	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = DefaultOrderPageSize
	}
	if pageSize < 1 || pageSize > MaxOrderPageSize {
		return nil, fmt.Errorf("list orders: page size must be between 1 and %d", MaxOrderPageSize)
	}

	// Guard the offset calculation against integer overflow.
	if input.Page-1 > math.MaxInt/pageSize {
		return nil, fmt.Errorf("list orders: requested page is too large")
	}
	offset := (input.Page - 1) * pageSize

	var (
		orders []*entities.Order
		total  int64
		err    error
	)

	// A provided StoreID selects store scope; otherwise use customer scope.
	if input.StoreID != uuid.Nil {
		orders, total, err = uc.orders.ListByStoreID(ctx, input.StoreID, offset, pageSize)
	} else {
		orders, total, err = uc.orders.ListByCustomerID(ctx, input.UserID, offset, pageSize)
	}
	if err != nil {
		uc.log(ctx, "list_orders", "load_orders")
		return nil, fmt.Errorf("list orders: load orders: %w", err)
	}
	if total < 0 {
		uc.log(ctx, "list_orders", "invalid_repository_result")
		return nil, fmt.Errorf("list orders: repository returned a negative total")
	}
	if len(orders) > pageSize {
		uc.log(ctx, "list_orders", "invalid_repository_result")
		return nil, fmt.Errorf("list orders: repository returned more orders than requested")
	}

	// A valid empty page is still a successful result.
	if len(orders) == 0 {
		uc.log(ctx, "list_orders", "")
		return &OrderPage{
			Items: []*OrderDetails{}, Page: input.Page,
			PageSize: pageSize, TotalItems: total,
			TotalPages: totalPages(total, pageSize),
		}, nil
	}

	orderIDs := make([]uuid.UUID, 0, len(orders))
	allowedOrderIDs := make(map[uuid.UUID]struct{}, len(orders))
	seenOrderIDs := make(map[uuid.UUID]struct{}, len(orders))

	for _, order := range orders {
		if order == nil || order.ID == uuid.Nil {
			uc.log(ctx, "list_orders", "invalid_repository_result")
			return nil, fmt.Errorf("list orders: repository returned an invalid order")
		}

		// Verify scope even though the repository is expected to filter by it.
		if input.StoreID != uuid.Nil {
			if order.StoreID != input.StoreID {
				uc.log(ctx, "list_orders", "scope_mismatch")
				return nil, fmt.Errorf("list orders: repository returned an order outside store scope")
			}
		} else if order.CustomerID != input.UserID {
			uc.log(ctx, "list_orders", "scope_mismatch")
			return nil, fmt.Errorf("list orders: repository returned an order outside customer scope")
		}

		if _, duplicate := seenOrderIDs[order.ID]; duplicate {
			uc.log(ctx, "list_orders", "invalid_repository_result")
			return nil, fmt.Errorf("list orders: repository returned a duplicate order ID")
		}
		seenOrderIDs[order.ID] = struct{}{}
		allowedOrderIDs[order.ID] = struct{}{}
		orderIDs = append(orderIDs, order.ID)
	}

	lines, err := uc.products.FindByOrderIDs(ctx, orderIDs)
	if err != nil {
		uc.log(ctx, "list_orders", "load_order_products")
		return nil, fmt.Errorf("list orders: load order products: %w", err)
	}

	if err := validateOrderProductLines(lines, allowedOrderIDs); err != nil {
		uc.log(ctx, "list_orders", "invalid_repository_result")
		return nil, fmt.Errorf("list orders: invalid order products: %w", err)
	}

	linesByOrder := make(map[uuid.UUID][]*entities.OrderProduct, len(orders))
	for _, line := range lines {
		linesByOrder[line.OrderID] = append(linesByOrder[line.OrderID], line)
	}

	items := make([]*OrderDetails, 0, len(orders))
	for _, order := range orders {
		items = append(items, &OrderDetails{
			Order: order, Products: linesByOrder[order.ID],
		})
	}

	uc.log(ctx, "list_orders", "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name: "order_list_requests", Value: 1,
		Labels: map[string]string{"operation": "list_orders"},
	})

	return &OrderPage{
		Items: items, Page: input.Page, PageSize: pageSize,
		TotalItems: total, TotalPages: totalPages(total, pageSize),
	}, nil
}

// totalPages calculates the number of pages without overflowing total+pageSize.
func totalPages(total int64, pageSize int) int64 {
	if total <= 0 {
		return 0
	}
	size := int64(pageSize)
	return 1 + (total-1)/size
}

// log records listing outcomes without changing the operation's result.
func (uc *ListOrders) log(ctx context.Context, operation, failureCategory string) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event: "order.retrieval", Operation: operation, FailureCategory: failureCategory,
	})
}
