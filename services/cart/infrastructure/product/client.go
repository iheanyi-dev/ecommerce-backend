// Package product implements the Cart service's Product gRPC adapter.
package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
)

// Client implements ports.ProductClient.
type Client struct {
	conn          *grpc.ClientConn
	client        productv1.ProductServiceClient
	serviceName   string
	serviceSecret string
}

var _ ports.ProductClient = (*Client)(nil)

// NewClient creates the Product gRPC client and registers connection cleanup.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("Product client: config is required")
	}
	if cfg.ProductServiceAddr == "" {
		return nil, errors.New("Product client: PRODUCT_SERVICE_ADDR is required")
	}
	if cfg.ServiceAuthName == "" || cfg.ServiceAuthSecret == "" {
		return nil, errors.New("Product client: service authentication configuration is required")
	}

	conn, err := grpc.NewClient(
		cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Product gRPC connection: %w", err)
	}

	client := NewClientWithStub(
		productv1.NewProductServiceClient(conn),
		cfg.ServiceAuthName,
		cfg.ServiceAuthSecret,
	)
	client.conn = conn

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})

	return client, nil
}

// NewClientWithStub permits deterministic testing with an injected gRPC stub.
func NewClientWithStub(
	client productv1.ProductServiceClient,
	serviceName string,
	serviceSecret string,
) *Client {
	return &Client{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

// GetProductsByIDs retrieves authoritative product snapshots.
// Products absent from a successful response are treated as deleted or missing.
// RPC errors are returned as errors and never treated as missing products.
func (c *Client) GetProductsByIDs(
	ctx context.Context,
	productIDs []uuid.UUID,
) ([]ports.ProductSnapshot, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("get Products by IDs: Product client is not initialized")
	}

	if len(productIDs) == 0 {
		return []ports.ProductSnapshot{}, nil
	}

	requestIDs := make([]string, 0, len(productIDs))
	for _, id := range productIDs {
		requestIDs = append(requestIDs, id.String())
	}

	response, err := c.client.GetProductsByIDs(
		c.authenticatedContext(ctx),
		&productv1.GetProductsByIDsRequest{
			ProductIds: requestIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get Products by IDs: %w", err)
	}
	if response == nil {
		return nil, errors.New("get Products by IDs: Product service returned an empty response")
	}

	snapshots := make([]ports.ProductSnapshot, 0, len(response.GetProducts()))
	for _, item := range response.GetProducts() {
		if item == nil {
			return nil, errors.New("get Products by IDs: Product service returned a nil product")
		}

		productID, err := uuid.Parse(item.GetProductId())
		if err != nil {
			return nil, fmt.Errorf(
				"get Products by IDs: Product service returned invalid product ID %q: %w",
				item.GetProductId(),
				err,
			)
		}

		storeID, err := uuid.Parse(item.GetStoreId())
		if err != nil {
			return nil, fmt.Errorf(
				"get Products by IDs: Product %s has invalid Store ID %q: %w",
				productID,
				item.GetStoreId(),
				err,
			)
		}

		snapshots = append(snapshots, ports.ProductSnapshot{
			ID:           productID,
			StoreID:      storeID,
			Name:         item.GetName(),
			Description:  item.GetDescription(),
			PriceMinor:   item.GetPriceMinor(),
			Currency:     item.GetCurrency(),
			Quantity:     item.GetQuantity(),
			HasPromotion: item.GetHasPromotion(),
		})
	}

	return snapshots, nil
}

// authenticatedContext adds service authentication and propagates trace context.
func (c *Client) authenticatedContext(ctx context.Context) context.Context {
	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = metadata.MD{}
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(md))

	return metadata.NewOutgoingContext(ctx, md)
}
