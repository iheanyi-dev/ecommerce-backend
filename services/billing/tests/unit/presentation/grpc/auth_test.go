package grpc_test

import (
	"context"
	"testing"

	presentationgrpc "github.com/iheanyi-dev/ecommerce-backend/services/billing/presentation/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServiceAuthInterceptor(t *testing.T) {
	tests := []struct {
		name    string
		service string
		secret  string
		wantErr bool
	}{
		{"valid credentials", "billing", "secret", false},
		{"missing credentials", "", "", true},
		{"invalid service name", "payment", "secret", true},
		{"invalid secret", "billing", "wrong", true},
		{"missing secret", "billing", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := presentationgrpc.NewServiceAuthInterceptor("billing", "secret")
			ctx := metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-service-name", tt.service,
					"x-service-secret", tt.secret,
				),
			)

			called := false
			handler := func(context.Context, any) (any, error) {
				called = true
				return "ok", nil
			}

			result, err := interceptor(
				ctx,
				"request",
				&grpc.UnaryServerInfo{FullMethod: "/billing.v1.BillingService/GetPlan"},
				handler,
			)

			if tt.wantErr {
				if status.Code(err) != codes.Unauthenticated {
					t.Fatalf("expected Unauthenticated, got %v", err)
				}
				if called {
					t.Fatal("handler was called with invalid credentials")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != "ok" || !called {
				t.Fatal("handler was not called successfully")
			}
		})
	}
}
