package identity_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	identitygrpc "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/identity"
)

const bufSize = 1024 * 1024

type mockGetUserService struct {
	execute func(context.Context, string) (*dto.GetUserResult, error)
}

func (m *mockGetUserService) Execute(
	ctx context.Context,
	userID string,
) (*dto.GetUserResult, error) {
	return m.execute(ctx, userID)
}

var _ ports.GetUserService = (*mockGetUserService)(nil)

type mockPromoteUserToVendorService struct{}

func (m *mockPromoteUserToVendorService) Execute(
	context.Context,
	string,
) error {
	return nil
}

var _ ports.PromoteUserToVendorService = (*mockPromoteUserToVendorService)(nil)

type recordingTracer struct {
	started chan context.Context
}

func (t *recordingTracer) Start(
	ctx context.Context,
	_ string,
) (context.Context, ports.Span) {
	t.started <- ctx

	return ctx, recordingSpan{}
}

type recordingSpan struct{}

func (recordingSpan) End() {}

var _ ports.Tracer = (*recordingTracer)(nil)

func TestProvider_TraceContextCrossesGRPCBoundary(t *testing.T) {
	traceProvider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		require.NoError(t, traceProvider.Shutdown(context.Background()))
	})

	originalTracerProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		otel.SetTracerProvider(originalTracerProvider)
		otel.SetTextMapPropagator(originalPropagator)
	})

	tracer := &recordingTracer{
		started: make(chan context.Context, 1),
	}

	identityServer := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(ctx context.Context, userID string) (*dto.GetUserResult, error) {
				require.True(
					t,
					oteltrace.SpanContextFromContext(ctx).IsValid(),
					"Identity handler should receive the distributed trace context",
				)

				return &dto.GetUserResult{
					ID:       userID,
					Email:    "vendor@example.com",
					FullName: "Vendor User",
					Role:     "vendor",
					Status:   "active",
				}, nil
			},
		},
		&mockPromoteUserToVendorService{},
	)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			identitygrpc.NewTracingInterceptor(tracer),
			identitygrpc.NewServiceAuthInterceptor(
				"store",
				"store-secret",
			),
		),
	)

	identityv1.RegisterIdentityServiceServer(grpcServer, identityServer)

	listener := bufconn.Listen(bufSize)
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	client := identityv1.NewIdentityServiceClient(conn)

	provider := identity.NewProvider(
		client,
		"store",
		"store-secret",
	)

	userID := uuid.New()

	clientCtx, clientSpan := traceProvider.Tracer("store").Start(
		context.Background(),
		"store.identity.get-user",
	)
	defer clientSpan.End()

	_, err = provider.IsVendor(clientCtx, userID)
	require.NoError(t, err)

	select {
	case identityCtx := <-tracer.started:
		identitySpanContext := oteltrace.SpanContextFromContext(identityCtx)
		clientSpanContext := oteltrace.SpanContextFromContext(clientCtx)

		require.True(t, identitySpanContext.IsValid())
		require.True(t, clientSpanContext.IsValid())
		require.Equal(
			t,
			clientSpanContext.TraceID(),
			identitySpanContext.TraceID(),
			"Identity should remain on the Store trace",
		)
		require.True(
			t,
			identitySpanContext.IsRemote(),
			"Identity should receive the Store trace context as a remote parent",
		)
		require.Equal(
			t,
			clientSpanContext.SpanID(),
			identitySpanContext.SpanID(),
			"Identity interceptor should receive the propagated Store span context",
		)

	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Identity tracing interceptor")
	}
}
