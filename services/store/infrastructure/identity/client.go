package identity

import (
	"context"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// NewClient creates the Identity Service gRPC client connection and client.
//
// The connection lifecycle is owned by Fx. Store uses insecure transport here
// because service-to-service TLS is an infrastructure/deployment concern and
// can be introduced without changing the application IdentityProvider port.
func NewClient(
	lc fx.Lifecycle,
	cfg *config.Config,
) (identityv1.IdentityServiceClient, error) {
	conn, err := grpc.NewClient(
		cfg.IdentityServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return conn.Close()
		},
	})

	return identityv1.NewIdentityServiceClient(conn), nil
}
