package embedding_test

import (
	"context"
	"math"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	embeddingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/embedding/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/embedding"
)

type embeddingServer struct {
	embeddingv1.UnimplementedEmbeddingServiceServer
}

func (s *embeddingServer) RetrieveSimilarProducts(
	ctx context.Context,
	req *embeddingv1.RetrieveSimilarProductsRequest,
) (*embeddingv1.RetrieveSimilarProductsResponse, error) {
	if req.GetQuery() != "laptop" || req.GetTopK() != 10 || req.GetOffset() != 20 {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "unexpected product request")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "missing metadata")
	}
	if md.Get("x-service-name")[0] != "chat" ||
		md.Get("x-service-secret")[0] != "secret" {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "invalid authentication metadata")
	}

	return &embeddingv1.RetrieveSimilarProductsResponse{
		Products: []*embeddingv1.SimilarProduct{
			{ProductId: "11111111-1111-4111-8111-111111111111", Score: 0.95},
		},
	}, nil
}

func (s *embeddingServer) RetrieveSimilarStores(
	ctx context.Context,
	req *embeddingv1.RetrieveSimilarStoresRequest,
) (*embeddingv1.RetrieveSimilarStoresResponse, error) {
	if req.GetQuery() != "electronics" || req.GetTopK() != 5 || req.GetOffset() != 10 {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "unexpected store request")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "missing metadata")
	}
	if md.Get("x-service-name")[0] != "chat" ||
		md.Get("x-service-secret")[0] != "secret" {
		return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "invalid authentication metadata")
	}

	return &embeddingv1.RetrieveSimilarStoresResponse{
		Stores: []*embeddingv1.SimilarStore{
			{StoreId: "22222222-2222-4222-8222-222222222222", Score: 0.85},
		},
	}, nil
}

func startEmbeddingServer(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	embeddingv1.RegisterEmbeddingServiceServer(server, &embeddingServer{})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func TestNewGRPCClientRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		serviceName string
		secret      string
		timeout     time.Duration
	}{
		{
			name:        "empty address",
			address:     " ",
			serviceName: "chat",
			secret:      "secret",
			timeout:     time.Second,
		},
		{
			name:        "empty service name",
			address:     "localhost:8092",
			serviceName: " ",
			secret:      "secret",
			timeout:     time.Second,
		},
		{
			name:        "empty secret",
			address:     "localhost:8092",
			serviceName: "chat",
			secret:      "",
			timeout:     time.Second,
		},
		{
			name:        "non-positive timeout",
			address:     "localhost:8092",
			serviceName: "chat",
			secret:      "secret",
			timeout:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := embedding.NewGRPCClient(
				tt.address,
				tt.serviceName,
				tt.secret,
				tt.timeout,
			)
			if err == nil {
				if client != nil {
					_ = client.Close()
				}
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestGRPCClientRetrievesSimilarProductsAndStores(t *testing.T) {
	address := startEmbeddingServer(t)

	client, err := embedding.NewGRPCClient(
		address,
		"chat",
		"secret",
		5*time.Second,
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	productCandidates, err := client.FindSimilarProducts(
		context.Background(),
		"laptop",
		10,
		20,
	)
	if err != nil {
		t.Fatalf("find similar products: %v", err)
	}

	if len(productCandidates) != 1 {
		t.Fatalf("expected 1 product candidate, got %d", len(productCandidates))
	}

	expectedProductID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	if productCandidates[0].ID != expectedProductID {
		t.Fatalf("expected product ID %s, got %s", expectedProductID, productCandidates[0].ID)
	}
	if math.Abs(productCandidates[0].Score-0.95) > 1e-6 {
		t.Fatalf("expected product score 0.95, got %f", productCandidates[0].Score)
	}

	storeCandidates, err := client.FindSimilarStores(
		context.Background(),
		"electronics",
		5,
		10,
	)
	if err != nil {
		t.Fatalf("find similar stores: %v", err)
	}

	if len(storeCandidates) != 1 {
		t.Fatalf("expected 1 store candidate, got %d", len(storeCandidates))
	}

	expectedStoreID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	if storeCandidates[0].ID != expectedStoreID {
		t.Fatalf("expected store ID %s, got %s", expectedStoreID, storeCandidates[0].ID)
	}
	if math.Abs(storeCandidates[0].Score-0.85) > 1e-6 {
		t.Fatalf("expected store score 0.85, got %f", storeCandidates[0].Score)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
}
