package embedding

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	embeddingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/embedding/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// GRPCClient retrieves semantic recommendation candidates from Embedding.
type GRPCClient struct {
	client   embeddingv1.EmbeddingServiceClient
	conn     *grpc.ClientConn
	metadata metadata.MD
	timeout  time.Duration
}

// NewGRPCClient creates an authenticated connection to Embedding.
func NewGRPCClient(
	address string,
	serviceName string,
	serviceSecret string,
	timeout time.Duration,
) (*GRPCClient, error) {
	address = strings.TrimSpace(address)
	serviceName = strings.TrimSpace(serviceName)

	if address == "" {
		return nil, fmt.Errorf("Embedding gRPC address must not be empty")
	}
	if serviceName == "" {
		return nil, fmt.Errorf("Embedding caller service name must not be empty")
	}
	if serviceSecret == "" {
		return nil, fmt.Errorf("Embedding caller service secret must not be empty")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("Embedding gRPC timeout must be positive")
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Embedding gRPC connection: %w", err)
	}

	return &GRPCClient{
		client: embeddingv1.NewEmbeddingServiceClient(conn),
		conn:   conn,
		metadata: metadata.Pairs(
			"x-service-name", serviceName,
			"x-service-secret", serviceSecret,
		),
		timeout: timeout,
	}, nil
}

// FindSimilarProducts retrieves a page of semantically similar Products.
func (c *GRPCClient) FindSimilarProducts(
	ctx context.Context,
	query string,
	topK int,
	offset int,
) ([]ports.RecommendationCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, c.metadata)

	response, err := c.client.RetrieveSimilarProducts(
		ctx,
		&embeddingv1.RetrieveSimilarProductsRequest{
			Query:  query,
			TopK:   int32(topK),
			Offset: int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("retrieve similar Products from Embedding: %w", err)
	}

	candidates := make([]ports.RecommendationCandidate, 0, len(response.GetProducts()))
	for _, product := range response.GetProducts() {
		productID, err := uuid.Parse(product.GetProductId())
		if err != nil {
			return nil, fmt.Errorf("parse recommended Product ID: %w", err)
		}

		candidates = append(candidates, ports.RecommendationCandidate{
			ID:    productID,
			Score: float64(product.GetScore()),
		})
	}

	return candidates, nil
}

// FindSimilarStores retrieves a page of semantically similar Stores.
func (c *GRPCClient) FindSimilarStores(
	ctx context.Context,
	query string,
	topK int,
	offset int,
) ([]ports.RecommendationCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, c.metadata)

	response, err := c.client.RetrieveSimilarStores(
		ctx,
		&embeddingv1.RetrieveSimilarStoresRequest{
			Query:  query,
			TopK:   int32(topK),
			Offset: int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("retrieve similar Stores from Embedding: %w", err)
	}

	candidates := make([]ports.RecommendationCandidate, 0, len(response.GetStores()))
	for _, store := range response.GetStores() {
		storeID, err := uuid.Parse(store.GetStoreId())
		if err != nil {
			return nil, fmt.Errorf("parse recommended Store ID: %w", err)
		}

		candidates = append(candidates, ports.RecommendationCandidate{
			ID:    storeID,
			Score: float64(store.GetScore()),
		})
	}

	return candidates, nil
}

// Close releases the underlying gRPC connection.
func (c *GRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}
