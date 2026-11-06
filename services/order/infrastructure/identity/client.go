// Package identity implements Order's outbound Identity service adapter.
package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client calls Identity's authenticated internal gRPC API.
type Client struct {
	client        identityv1.IdentityServiceClient
	conn          *grpc.ClientConn
	serviceName   string
	serviceSecret string
}

// NewClient creates the Identity gRPC connection and ties its lifetime to Order's Fx app.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("create Identity client: config is nil")
	}
	if strings.TrimSpace(cfg.IdentityServiceAddr) == "" {
		return nil, fmt.Errorf("create Identity client: IDENTITY_SERVICE_ADDR is required")
	}
	if strings.TrimSpace(cfg.OrderServiceAuthName) == "" ||
		strings.TrimSpace(cfg.OrderServiceAuthSecret) == "" {
		return nil, fmt.Errorf("create Identity client: Order service credentials are required")
	}

	conn, err := grpc.NewClient(
		cfg.IdentityServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Identity gRPC connection: %w", err)
	}

	client := &Client{
		client:        identityv1.NewIdentityServiceClient(conn),
		conn:          conn,
		serviceName:   cfg.OrderServiceAuthName,
		serviceSecret: cfg.OrderServiceAuthSecret,
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close Identity gRPC connection: %w", err)
			}
			return nil
		},
	})

	return client, nil
}

// GetUserEmail retrieves the email belonging to the supplied Identity user.
func (c *Client) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("get Identity user email: client is unavailable")
	}
	if userID == uuid.Nil {
		return "", fmt.Errorf("get Identity user email: user ID is required")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	response, err := c.client.GetUser(ctx, &identityv1.GetUserRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return "", fmt.Errorf("call Identity GetUser RPC: %w", err)
	}
	if response == nil || response.GetUser() == nil {
		return "", fmt.Errorf("Identity returned an empty GetUser response")
	}

	user := response.GetUser()
	returnedID, err := uuid.Parse(user.GetId())
	if err != nil || returnedID != userID {
		return "", fmt.Errorf("Identity returned an unexpected user ID")
	}

	email := strings.TrimSpace(user.GetEmail())
	if email == "" {
		return "", fmt.Errorf("Identity returned an empty user email")
	}

	return email, nil
}

var _ interface {
	GetUserEmail(context.Context, uuid.UUID) (string, error)
} = (*Client)(nil)
