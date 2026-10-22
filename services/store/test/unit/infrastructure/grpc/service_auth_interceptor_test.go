package grpc_test

import (
	"context"
	"testing"

	storegrpc "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServiceAuthInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configName string
		configKey  string
		metadata   metadata.MD
		wantCode   codes.Code
	}{
		{
			name:       "accepts valid credentials",
			configName: "product",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-name", "product",
				"x-service-secret", "test-service-secret",
			),
			wantCode: codes.OK,
		},
		{
			name:       "rejects missing metadata",
			configName: "product",
			configKey:  "test-service-secret",
			wantCode:   codes.Unauthenticated,
		},
		{
			name:       "rejects missing service name",
			configName: "product",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-secret", "test-service-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "rejects missing secret",
			configName: "product",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-name", "product",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "rejects wrong service name",
			configName: "product",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-name", "unknown",
				"x-service-secret", "test-service-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "rejects wrong secret",
			configName: "product",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-name", "product",
				"x-service-secret", "wrong-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "rejects empty configured service name",
			configName: "",
			configKey:  "test-service-secret",
			metadata: metadata.Pairs(
				"x-service-name", "product",
				"x-service-secret", "test-service-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "rejects empty configured secret",
			configName: "product",
			configKey:  "",
			metadata: metadata.Pairs(
				"x-service-name", "product",
				"x-service-secret", "test-service-secret",
			),
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			interceptor := storegrpc.NewServiceAuthInterceptor(
				tt.configName,
				tt.configKey,
			)

			ctx := context.Background()
			if tt.metadata != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.metadata)
			}

			handlerCalled := false
			_, err := interceptor(
				ctx,
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/store.v1.StoreService/Test"},
				func(context.Context, any) (any, error) {
					handlerCalled = true
					return "ok", nil
				},
			)

			if tt.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				if !handlerCalled {
					t.Fatal("expected handler to be called")
				}
				return
			}

			if status.Code(err) != tt.wantCode {
				t.Fatalf("expected %s, got %s: %v", tt.wantCode, status.Code(err), err)
			}
			if handlerCalled {
				t.Fatal("handler must not run for invalid credentials")
			}
		})
	}
}
