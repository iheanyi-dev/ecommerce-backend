package billing

import (
	"context"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// NewClient creates the Billing Service gRPC client and registers connection
// cleanup with the Store application lifecycle.
func NewClient(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
) (billingv1.BillingServiceClient, error) {
	connection, err := grpc.NewClient(
		cfg.BillingServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return connection.Close()
		},
	})

	return billingv1.NewBillingServiceClient(connection), nil
}
