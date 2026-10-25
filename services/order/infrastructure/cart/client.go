// Package cart implements Order's outbound Cart service adapter.
package cart

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	cartv1 "github.com/iheanyi-dev/ecommerce-backend/proto/cart/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client calls Cart's authenticated internal gRPC API.
type Client struct {
	client        cartv1.CartServiceClient
	conn          *grpc.ClientConn
	serviceName   string
	serviceSecret string
}

// NewClient creates the gRPC connection and ties its lifetime to Order's Fx app.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("create Cart client: config is nil")
	}
	if cfg.CartServiceAddr == "" {
		return nil, fmt.Errorf("create Cart client: CART_SERVICE_ADDR is required")
	}

	conn, err := grpc.NewClient(
		cfg.CartServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Cart gRPC connection: %w", err)
	}

	client := &Client{
		client:        cartv1.NewCartServiceClient(conn),
		conn:          conn,
		serviceName:   cfg.OrderServiceAuthName,
		serviceSecret: cfg.OrderServiceAuthSecret,
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close Cart gRPC connection: %w", err)
			}
			return nil
		},
	})

	return client, nil
}

// Checkout refreshes the customer's cart and converts Cart's response into
// the application port's checkout snapshot.
func (c *Client) Checkout(
	ctx context.Context,
	customerID, cartID uuid.UUID,
) (*ports.CartCheckoutOutput, error) {
	if customerID == uuid.Nil {
		return nil, fmt.Errorf("checkout Cart: customer ID is required")
	}
	if cartID == uuid.Nil {
		return nil, fmt.Errorf("checkout Cart: cart ID is required")
	}
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("checkout Cart: client is unavailable")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	response, err := c.client.CheckoutCartForOrder(ctx, &cartv1.CheckoutCartForOrderRequest{
		CustomerId: customerID.String(),
		CartId:     cartID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("checkout Cart through gRPC: %w", err)
	}

	return mapCheckoutResponse(response, cartID)
}

func mapCheckoutResponse(
	response *cartv1.CheckoutCartForOrderResponse,
	expectedCartID uuid.UUID,
) (*ports.CartCheckoutOutput, error) {
	if response == nil {
		return nil, fmt.Errorf("Cart returned a nil checkout response")
	}

	cartID, err := uuid.Parse(response.GetCartId())
	if err != nil || cartID == uuid.Nil {
		return nil, fmt.Errorf("Cart returned an invalid cart ID")
	}
	if cartID != expectedCartID {
		return nil, fmt.Errorf("Cart returned a different cart ID than requested")
	}

	output := &ports.CartCheckoutOutput{
		CartID:            cartID,
		Stores:            make([]ports.CartCheckoutStore, 0, len(response.GetStores())),
		RemovedProductIDs: make([]uuid.UUID, 0, len(response.GetRemovedProductIds())),
	}

	for _, store := range response.GetStores() {
		if store == nil {
			return nil, fmt.Errorf("Cart returned a nil store")
		}

		storeID, err := uuid.Parse(store.GetStoreId())
		if err != nil || storeID == uuid.Nil {
			return nil, fmt.Errorf("Cart returned an invalid store ID")
		}

		mappedStore := ports.CartCheckoutStore{
			StoreID: storeID,
			Items:   make([]ports.CartCheckoutItem, 0, len(store.GetItems())),
		}

		for _, item := range store.GetItems() {
			if item == nil {
				return nil, fmt.Errorf("Cart returned a nil checkout item")
			}

			productID, err := uuid.Parse(item.GetProductId())
			if err != nil || productID == uuid.Nil {
				return nil, fmt.Errorf("Cart returned an invalid product ID")
			}
			if item.GetQuantity() <= 0 || item.GetPriceMinor() < 0 || item.GetCurrency() == "" {
				return nil, fmt.Errorf("Cart returned invalid product checkout details")
			}

			mappedStore.Items = append(mappedStore.Items, ports.CartCheckoutItem{
				ProductID:  productID,
				Name:       item.GetName(),
				Quantity:   item.GetQuantity(),
				PriceMinor: item.GetPriceMinor(),
				Currency:   item.GetCurrency(),
			})
		}

		output.Stores = append(output.Stores, mappedStore)
	}

	for _, rawID := range response.GetRemovedProductIds() {
		productID, err := uuid.Parse(rawID)
		if err != nil || productID == uuid.Nil {
			return nil, fmt.Errorf("Cart returned an invalid removed product ID")
		}
		output.RemovedProductIDs = append(output.RemovedProductIDs, productID)
	}

	return output, nil
}
