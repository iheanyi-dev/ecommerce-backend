package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	identityinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/identity"
)

type mockIdentityClient struct {
	getUserFunc         func(context.Context, *identityv1.GetUserRequest, ...grpc.CallOption) (*identityv1.GetUserResponse, error)
	promoteToVendorFunc func(context.Context, *identityv1.PromoteToVendorRequest, ...grpc.CallOption) (*identityv1.PromoteToVendorResponse, error)

	getUserCalled         bool
	promoteToVendorCalled bool
}

func (m *mockIdentityClient) GetUser(
	ctx context.Context,
	req *identityv1.GetUserRequest,
	opts ...grpc.CallOption,
) (*identityv1.GetUserResponse, error) {
	m.getUserCalled = true

	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, req, opts...)
	}

	return nil, nil
}

func (m *mockIdentityClient) PromoteToVendor(
	ctx context.Context,
	req *identityv1.PromoteToVendorRequest,
	opts ...grpc.CallOption,
) (*identityv1.PromoteToVendorResponse, error) {
	m.promoteToVendorCalled = true

	if m.promoteToVendorFunc != nil {
		return m.promoteToVendorFunc(ctx, req, opts...)
	}

	return nil, nil
}

func (m *mockIdentityClient) mustEmbedUnimplementedIdentityServiceClient() {}

func TestProvider_IsVendor_ReturnsTrueForVendor(t *testing.T) {
	userID := uuid.New()

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			require.Equal(t, userID.String(), req.GetUserId())

			return &identityv1.GetUserResponse{
				User: &identityv1.User{
					Id:   userID.String(),
					Role: "vendor",
				},
			}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	isVendor, err := provider.IsVendor(context.Background(), userID)

	require.NoError(t, err)
	require.True(t, isVendor)
	require.True(t, client.getUserCalled)
}

func TestProvider_IsVendor_ReturnsFalseForNonVendor(t *testing.T) {
	userID := uuid.New()

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			return &identityv1.GetUserResponse{
				User: &identityv1.User{
					Id:   userID.String(),
					Role: "user",
				},
			}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	isVendor, err := provider.IsVendor(context.Background(), userID)

	require.NoError(t, err)
	require.False(t, isVendor)
}

func TestProvider_IsVendor_PropagatesRPCError(t *testing.T) {
	expectedErr := status.Error(codes.Internal, "identity unavailable")

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			return nil, expectedErr
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	isVendor, err := provider.IsVendor(
		context.Background(),
		uuid.New(),
	)

	require.False(t, isVendor)
	require.ErrorIs(t, err, expectedErr)
}

func TestProvider_IsVendor_RejectsNilUserResponse(t *testing.T) {
	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			return &identityv1.GetUserResponse{}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	isVendor, err := provider.IsVendor(
		context.Background(),
		uuid.New(),
	)

	require.False(t, isVendor)
	require.Error(t, err)
}

func TestProvider_PromoteToVendor_CallsIdentity(t *testing.T) {
	userID := uuid.New()

	client := &mockIdentityClient{
		promoteToVendorFunc: func(
			ctx context.Context,
			req *identityv1.PromoteToVendorRequest,
			_ ...grpc.CallOption,
		) (*identityv1.PromoteToVendorResponse, error) {
			require.Equal(t, userID.String(), req.GetUserId())

			return &identityv1.PromoteToVendorResponse{}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	err := provider.PromoteToVendor(
		context.Background(),
		userID,
	)

	require.NoError(t, err)
	require.True(t, client.promoteToVendorCalled)
}

func TestProvider_PromoteToVendor_PropagatesRPCError(t *testing.T) {
	expectedErr := errors.New("identity unavailable")

	client := &mockIdentityClient{
		promoteToVendorFunc: func(
			ctx context.Context,
			req *identityv1.PromoteToVendorRequest,
			_ ...grpc.CallOption,
		) (*identityv1.PromoteToVendorResponse, error) {
			return nil, expectedErr
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	err := provider.PromoteToVendor(
		context.Background(),
		uuid.New(),
	)

	require.ErrorIs(t, err, expectedErr)
}

func TestProvider_AttachesServiceAuthenticationMetadata(t *testing.T) {
	userID := uuid.New()

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)
			require.Equal(t, []string{"store"}, md.Get("x-service-name"))
			require.Equal(t, []string{"secret"}, md.Get("x-service-secret"))

			return &identityv1.GetUserResponse{
				User: &identityv1.User{
					Id:   userID.String(),
					Role: "vendor",
				},
			}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	_, err := provider.IsVendor(
		context.Background(),
		userID,
	)

	require.NoError(t, err)
}

func TestProvider_ForwardsRequestIDMetadata(t *testing.T) {
	userID := uuid.New()

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)
			require.Equal(t, []string{"request-123"}, md.Get("x-request-id"))

			return &identityv1.GetUserResponse{
				User: &identityv1.User{
					Id:   userID.String(),
					Role: "vendor",
				},
			}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("x-request-id", "request-123"),
	)

	_, err := provider.IsVendor(ctx, userID)

	require.NoError(t, err)
}

func TestProvider_InjectsTraceContextIntoOutgoingMetadata(t *testing.T) {
	userID := uuid.New()

	tracerProvider := trace.NewTracerProvider()
	t.Cleanup(func() {
		require.NoError(t, tracerProvider.Shutdown(context.Background()))
	})

	originalTracerProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		otel.SetTracerProvider(originalTracerProvider)
		otel.SetTextMapPropagator(originalPropagator)
	})

	tracer := otel.Tracer("store-test")
	ctx, span := tracer.Start(context.Background(), "store.identity.test")
	defer span.End()

	client := &mockIdentityClient{
		getUserFunc: func(
			ctx context.Context,
			req *identityv1.GetUserRequest,
			_ ...grpc.CallOption,
		) (*identityv1.GetUserResponse, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)

			require.NotEmpty(t, md.Get("traceparent"))

			return &identityv1.GetUserResponse{
				User: &identityv1.User{
					Id:   userID.String(),
					Role: "vendor",
				},
			}, nil
		},
	}

	provider := identityinfra.NewProvider(
		client,
		"store",
		"secret",
	)

	_, err := provider.IsVendor(ctx, userID)

	require.NoError(t, err)
}
