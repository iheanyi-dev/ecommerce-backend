package product

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

func NewClient(
	lc fx.Lifecycle,
	cfg *config.Config,
) (ports.ProductCountProvider, error) {
	if cfg.ProductServiceAddr == "" {
		return nil, fmt.Errorf("PRODUCT_SERVICE_ADDR is required")
	}
	if cfg.ProductServiceName == "" {
		return nil, fmt.Errorf("PRODUCT_SERVICE_NAME is required")
	}
	if cfg.ProductServiceSecret == "" {
		return nil, fmt.Errorf("PRODUCT_SERVICE_SECRET is required")
	}

	conn, err := grpc.NewClient(
		cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Product Service gRPC client: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})

	return newProvider(
		productv1.NewProductServiceClient(conn),
		cfg.ProductServiceName,
		cfg.ProductServiceSecret,
	), nil
}

type provider struct {
	client        productv1.ProductServiceClient
	serviceName   string
	serviceSecret string
}

func newProvider(
	client productv1.ProductServiceClient,
	serviceName string,
	serviceSecret string,
) ports.ProductCountProvider {
	return &provider{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

func (p *provider) CountByStoreID(
	ctx context.Context,
	storeID uuid.UUID,
) (int, error) {
	if storeID == uuid.Nil {
		return 0, fmt.Errorf("count products: store ID is required")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", p.serviceName,
		"x-service-secret", p.serviceSecret,
	)

	response, err := p.client.CountProductsByStore(
		ctx,
		&productv1.CountProductsByStoreRequest{
			StoreId: storeID.String(),
		},
	)
	if err != nil {
		return 0, fmt.Errorf("count products for Store %s: %w", storeID, err)
	}
	if response == nil {
		return 0, fmt.Errorf(
			"count products for Store %s: empty response",
			storeID,
		)
	}

	count := response.GetCount()
	if count < 0 {
		return 0, fmt.Errorf(
			"count products for Store %s: negative count %d",
			storeID,
			count,
		)
	}
	if uint64(count) > uint64(math.MaxInt) {
		return 0, fmt.Errorf(
			"count products for Store %s: count %d exceeds int range",
			storeID,
			count,
		)
	}

	return int(count), nil
}
