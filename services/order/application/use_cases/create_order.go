package use_cases

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

type CreateOrderInput struct {
	CartID           uuid.UUID
	CustomerID       uuid.UUID
	DeliveryLocation string
}

type CreateOrder struct {
	cartClient ports.CartClient
	unitOfWork ports.OrderUnitOfWork
	logger     ports.Logger
	metrics    ports.Metrics
}

func NewCreateOrder(
	cartClient ports.CartClient,
	unitOfWork ports.OrderUnitOfWork,
	logger ports.Logger,
	metrics ports.Metrics,
) *CreateOrder {
	return &CreateOrder{
		cartClient: cartClient,
		unitOfWork: unitOfWork,
		logger:     logger,
		metrics:    metrics,
	}
}

func (uc *CreateOrder) Execute(
	ctx context.Context,
	input CreateOrderInput,
) ([]*entities.Order, error) {
	if input.CartID == uuid.Nil || input.CustomerID == uuid.Nil {
		return nil, fmt.Errorf("create order: cart ID and customer ID are required")
	}
	if strings.TrimSpace(input.DeliveryLocation) == "" {
		return nil, fmt.Errorf("create order: delivery location is required")
	}

	checkout, err := uc.cartClient.Checkout(ctx, input.CustomerID, input.CartID)
	if err != nil {
		uc.log(ctx, "order.create.failed", "cart_checkout")
		return nil, fmt.Errorf("create order: checkout cart: %w", err)
	}
	if checkout == nil {
		return nil, fmt.Errorf("create order: cart checkout returned no result")
	}
	if checkout.CartID != input.CartID {
		return nil, fmt.Errorf("create order: checkout cart ID does not match requested cart")
	}
	if len(checkout.Stores) == 0 {
		return nil, fmt.Errorf("create order: checkout contains no products")
	}

	orders := make([]*entities.Order, 0, len(checkout.Stores))
	orderProducts := make([]*entities.OrderProduct, 0)
	seenStores := make(map[uuid.UUID]struct{}, len(checkout.Stores))

	for _, store := range checkout.Stores {
		if store.StoreID == uuid.Nil {
			return nil, fmt.Errorf("create order: invalid store ID")
		}
		if _, exists := seenStores[store.StoreID]; exists {
			return nil, fmt.Errorf("create order: duplicate store groups")
		}
		seenStores[store.StoreID] = struct{}{}

		if len(store.Items) == 0 {
			return nil, fmt.Errorf("create order: store %s has no products", store.StoreID)
		}

		currency := strings.ToUpper(strings.TrimSpace(store.Items[0].Currency))
		if currency == "" {
			return nil, fmt.Errorf("create order: currency is required")
		}

		var subtotal int64
		for _, item := range store.Items {
			if item.ProductID == uuid.Nil || strings.TrimSpace(item.Name) == "" {
				return nil, fmt.Errorf("create order: invalid product snapshot")
			}
			if item.Quantity <= 0 {
				return nil, fmt.Errorf("create order: product quantity must be positive")
			}
			if item.PriceMinor < 0 {
				return nil, fmt.Errorf("create order: product price cannot be negative")
			}
			if !strings.EqualFold(currency, strings.TrimSpace(item.Currency)) {
				return nil, fmt.Errorf("create order: products within a store must use the same currency")
			}
			if item.PriceMinor != 0 && item.Quantity > math.MaxInt64/item.PriceMinor {
				return nil, fmt.Errorf("create order: product subtotal overflow")
			}

			lineTotal := item.PriceMinor * item.Quantity
			if subtotal > math.MaxInt64-lineTotal {
				return nil, fmt.Errorf("create order: store subtotal overflow")
			}
			subtotal += lineTotal
		}

		order, err := entities.NewOrder(
			store.StoreID,
			input.CustomerID,
			input.CartID,
			input.DeliveryLocation,
			currency,
			subtotal,
		)
		if err != nil {
			return nil, fmt.Errorf("create order: validate order: %w", err)
		}
		orders = append(orders, order)

		for _, item := range store.Items {
			product, err := entities.NewOrderProduct(
				uuid.New(),
				order.ID,
				item.ProductID,
				item.Name,
				item.Quantity,
				item.PriceMinor,
			)
			if err != nil {
				return nil, fmt.Errorf("create order: validate order product: %w", err)
			}
			productCopy := product
			orderProducts = append(orderProducts, &productCopy)
		}
	}

	err = uc.unitOfWork.WithinTransaction(ctx, func(
		orderRepo ports.OrderRepository,
		productRepo ports.OrderProductRepository,
	) error {
		if err := orderRepo.CreateMany(ctx, orders); err != nil {
			return fmt.Errorf("persist orders: %w", err)
		}
		if err := productRepo.CreateMany(ctx, orderProducts); err != nil {
			return fmt.Errorf("persist order products: %w", err)
		}
		return nil
	})
	if err != nil {
		uc.log(ctx, "order.create.failed", "persistence")
		return nil, fmt.Errorf("create order: transaction failed: %w", err)
	}

	uc.log(ctx, "order.create.succeeded", "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name: "orders_created", Value: float64(len(orders)),
		Labels: map[string]string{"operation": "create_order"},
	})
	_ = uc.metrics.Observe(ctx, ports.Metric{
		Name: "order_products_created", Value: float64(len(orderProducts)),
		Labels: map[string]string{"operation": "create_order"},
	})

	return orders, nil
}

func (uc *CreateOrder) log(ctx context.Context, event, failureCategory string) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event: event, Operation: "create_order", FailureCategory: failureCategory,
	})
}
