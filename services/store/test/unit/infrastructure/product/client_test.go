package product_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	productinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/product"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func (l *testLifecycle) stop(t *testing.T) {
	t.Helper()
	for i := len(l.hooks) - 1; i >= 0; i-- {
		if l.hooks[i].OnStop != nil {
			require.NoError(t, l.hooks[i].OnStop(context.Background()))
		}
	}
}

type productServer struct {
	productv1.UnimplementedProductServiceServer
	count int64
	err   error
	md    metadata.MD
}

func (s *productServer) CountProductsByStore(
	ctx context.Context,
	request *productv1.CountProductsByStoreRequest,
) (*productv1.CountProductsByStoreResponse, error) {
	s.md, _ = metadata.FromIncomingContext(ctx)

	if s.err != nil {
		return nil, s.err
	}

	if _, err := uuid.Parse(request.GetStoreId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid store ID")
	}

	return &productv1.CountProductsByStoreResponse{Count: s.count}, nil
}

func startTestClient(
	t *testing.T,
	count int64,
	rpcErr error,
) (*productServer, interface {
	CountByStoreID(context.Context, uuid.UUID) (int, error)
}) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	productService := &productServer{count: count, err: rpcErr}
	productv1.RegisterProductServiceServer(server, productService)

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	lifecycle := &testLifecycle{}
	cfg := &config.Config{
		ProductServiceAddr:   listener.Addr().String(),
		ProductServiceName:   "store",
		ProductServiceSecret: "shared-secret",
	}

	provider, err := productinfra.NewClient(lifecycle, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { lifecycle.stop(t) })

	return productService, provider
}

func TestCountByStoreID(t *testing.T) {
	storeID := uuid.New()
	server, provider := startTestClient(t, 3, nil)

	count, err := provider.CountByStoreID(context.Background(), storeID)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, []string{"store"}, server.md.Get("x-service-name"))
	require.Equal(t, []string{"shared-secret"}, server.md.Get("x-service-secret"))
}

func TestCountByStoreID_RejectsNilStoreID(t *testing.T) {
	_, provider := startTestClient(t, 3, nil)

	count, err := provider.CountByStoreID(context.Background(), uuid.Nil)

	require.ErrorContains(t, err, "store ID is required")
	require.Zero(t, count)
}

func TestCountByStoreID_PropagatesRPCError(t *testing.T) {
	_, provider := startTestClient(
		t,
		0,
		status.Error(codes.Unavailable, "service unavailable"),
	)

	count, err := provider.CountByStoreID(context.Background(), uuid.New())

	require.ErrorContains(t, err, "service unavailable")
	require.Zero(t, count)
}

func TestCountByStoreID_RejectsNegativeCount(t *testing.T) {
	_, provider := startTestClient(t, -1, nil)

	count, err := provider.CountByStoreID(context.Background(), uuid.New())

	require.ErrorContains(t, err, "negative count")
	require.Zero(t, count)
}

func TestCountByStoreID_UsesRequestContext(t *testing.T) {
	_, provider := startTestClient(t, 2, errors.New("unused"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := provider.CountByStoreID(ctx, uuid.New())

	require.Error(t, err)
	require.Zero(t, count)
}
