package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServiceAuthInterceptor(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		configKey  string
		metadata   metadata.MD
		wantCode   codes.Code
	}{
		{
			name:       "valid credentials",
			configName: "store",
			configKey:  "strong-secret",
			metadata: metadata.Pairs(
				"x-service-name", "store",
				"x-service-secret", "strong-secret",
			),
			wantCode: codes.OK,
		},
		{
			name:       "missing credentials",
			configName: "store",
			configKey:  "strong-secret",
			wantCode:   codes.Unauthenticated,
		},
		{
			name:       "wrong service name",
			configName: "store",
			configKey:  "strong-secret",
			metadata: metadata.Pairs(
				"x-service-name", "gateway",
				"x-service-secret", "strong-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "wrong secret",
			configName: "store",
			configKey:  "strong-secret",
			metadata: metadata.Pairs(
				"x-service-name", "store",
				"x-service-secret", "wrong-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "duplicate service name",
			configName: "store",
			configKey:  "strong-secret",
			metadata: metadata.Pairs(
				"x-service-name", "store",
				"x-service-name", "store",
				"x-service-secret", "strong-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "server credentials not configured",
			configName: "store",
			metadata: metadata.Pairs(
				"x-service-name", "store",
				"x-service-secret", "strong-secret",
			),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.metadata != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.metadata)
			}

			called := false
			interceptor := NewServiceAuthInterceptor(tt.configName, tt.configKey)
			_, err := interceptor(
				ctx,
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/order.v1.OrderService/Test"},
				func(context.Context, any) (any, error) {
					called = true
					return "ok", nil
				},
			)

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("status.Code(err) = %v, want %v (err: %v)", got, tt.wantCode, err)
			}
			if called != (tt.wantCode == codes.OK) {
				t.Errorf("handler called = %v, want %v", called, tt.wantCode == codes.OK)
			}
		})
	}
}
