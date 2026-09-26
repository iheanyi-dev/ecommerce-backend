package grpc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	identitygrpc "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/grpc"
)

func TestServiceAuthInterceptor_AllowsValidCredentials(t *testing.T) {
	interceptor := identitygrpc.NewServiceAuthInterceptor(
		"store-service",
		"super-secret",
	)

	called := false

	_, err := interceptor(
		contextWithServiceCredentials(
			context.Background(),
			"store-service",
			"super-secret",
		),
		nil,
		nil,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "ok", nil
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestServiceAuthInterceptor_RejectsMissingCredentials(t *testing.T) {
	testServiceAuthFailure(t, context.Background())
}

func TestServiceAuthInterceptor_RejectsInvalidServiceName(t *testing.T) {
	testServiceAuthFailure(
		t,
		contextWithServiceCredentials(
			context.Background(),
			"wrong-service",
			"super-secret",
		),
	)
}

func TestServiceAuthInterceptor_RejectsInvalidServiceSecret(t *testing.T) {
	testServiceAuthFailure(
		t,
		contextWithServiceCredentials(
			context.Background(),
			"store-service",
			"wrong-secret",
		),
	)
}

func TestServiceAuthInterceptor_RejectsEmptyConfiguredCredentials(t *testing.T) {
	interceptor := identitygrpc.NewServiceAuthInterceptor(
		"",
		"",
	)

	_, err := interceptor(
		contextWithServiceCredentials(
			context.Background(),
			"store-service",
			"super-secret",
		),
		nil,
		nil,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		},
	)

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf(
			"expected Unauthenticated, got %s",
			status.Code(err),
		)
	}
}

func testServiceAuthFailure(t *testing.T, ctx context.Context) {
	interceptor := identitygrpc.NewServiceAuthInterceptor(
		"store-service",
		"super-secret",
	)

	called := false

	_, err := interceptor(
		ctx,
		nil,
		nil,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "ok", nil
		},
	)

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf(
			"expected Unauthenticated, got %s",
			status.Code(err),
		)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func contextWithServiceCredentials(
	ctx context.Context,
	serviceName string,
	serviceSecret string,
) context.Context {
	return metadata.NewIncomingContext(
		ctx,
		metadata.Pairs(
			"x-service-name",
			serviceName,
			"x-service-secret",
			serviceSecret,
		),
	)
}

var _ grpc.UnaryServerInterceptor = identitygrpc.NewServiceAuthInterceptor(
	"service",
	"secret",
)
