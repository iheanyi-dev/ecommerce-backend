// Package store implements the Product service's Store gRPC adapter.
package store

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	storev1 "github.com/iheanyi-dev/ecommerce-backend/proto/store/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

// Client implements ports.StoreClient.
type Client struct {
	conn          *grpc.ClientConn
	client        storev1.StoreServiceClient
	serviceName   string
	serviceSecret string
}

var _ ports.StoreClient = (*Client)(nil)

// NewClient creates the Store gRPC client and registers connection cleanup.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("Store client: config is required")
	}
	if cfg.StoreServiceAddr == "" {
		return nil, errors.New("Store client: STORE_SERVICE_ADDR is required")
	}
	if cfg.ServiceAuthName == "" || cfg.ServiceAuthSecret == "" {
		return nil, errors.New("Store client: service authentication configuration is required")
	}

	conn, err := grpc.NewClient(
		cfg.StoreServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Store gRPC connection: %w", err)
	}

	client := NewClientWithStub(
		storev1.NewStoreServiceClient(conn),
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
	client storev1.StoreServiceClient,
	serviceName string,
	serviceSecret string,
) *Client {
	return &Client{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

// GetStoreIDByOwnerID returns the owner's store ID. NotFound means the owner
// has no store, matching the StoreClient port contract.
func (c *Client) GetStoreIDByOwnerID(ctx context.Context, ownerID string) (string, error) {
	ctx = c.authenticatedContext(ctx)
	response, err := c.client.GetStoreIDByOwnerID(ctx, &storev1.GetStoreIDByOwnerIDRequest{
		OwnerId: ownerID,
	})
	if status.Code(err) == codes.NotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get Store ID for owner %q: %w", ownerID, err)
	}
	if response == nil {
		return "", errors.New("get Store ID: Store service returned an empty response")
	}
	return response.GetStoreId(), nil
}

// IncrementProductCount atomically reserves one product slot through Store.
func (c *Client) IncrementProductCount(ctx context.Context, storeID string) error {
	_, err := c.client.IncrementProductCount(
		c.authenticatedContext(ctx),
		&storev1.IncrementProductCountRequest{StoreId: storeID},
	)
	if err != nil {
		return fmt.Errorf("reserve product capacity for Store %q: %w", storeID, err)
	}
	return nil
}

// DecrementProductCount releases a product slot previously reserved through Store.
func (c *Client) DecrementProductCount(ctx context.Context, storeID string) error {
	_, err := c.client.DecrementProductCount(
		c.authenticatedContext(ctx),
		&storev1.DecrementProductCountRequest{StoreId: storeID},
	)
	if err != nil {
		return fmt.Errorf("release product capacity for Store %q: %w", storeID, err)
	}
	return nil
}

// IsStoreActive reports whether Store currently allows customer access.
func (c *Client) IsStoreActive(ctx context.Context, storeID string) (bool, error) {
	response, err := c.client.IsStoreActive(
		c.authenticatedContext(ctx),
		&storev1.IsStoreActiveRequest{StoreId: storeID},
	)
	if err != nil {
		return false, fmt.Errorf("check Store %q status: %w", storeID, err)
	}
	if response == nil {
		return false, errors.New("check Store status: Store service returned an empty response")
	}
	return response.GetActive(), nil
}

// FilterActiveStoreIDs returns only active IDs according to Store.
func (c *Client) FilterActiveStoreIDs(ctx context.Context, storeIDs []string) ([]string, error) {
	if len(storeIDs) == 0 {
		return []string{}, nil
	}

	response, err := c.client.FilterActiveStoreIDs(
		c.authenticatedContext(ctx),
		&storev1.FilterActiveStoreIDsRequest{StoreIds: storeIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("filter active Store IDs: %w", err)
	}
	if response == nil {
		return nil, errors.New("filter active Store IDs: Store service returned an empty response")
	}
	return response.GetActiveStoreIds(), nil
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
