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
		name           string
		expectedName   string
		expectedSecret string
		ctx            context.Context
		want           codes.Code
	}{
		{
			name:           "valid credentials",
			expectedName:   "order",
			expectedSecret: "correct-secret",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"x-service-name", "order",
				"x-service-secret", "correct-secret",
			)),
			want: codes.OK,
		},
		{
			name:           "missing metadata",
			expectedName:   "order",
			expectedSecret: "correct-secret",
			ctx:            context.Background(),
			want:           codes.Unauthenticated,
		},
		{
			name:           "wrong secret",
			expectedName:   "order",
			expectedSecret: "correct-secret",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"x-service-name", "order",
				"x-service-secret", "wrong-secret",
			)),
			want: codes.Unauthenticated,
		},
		{
			name:           "duplicate secret",
			expectedName:   "order",
			expectedSecret: "correct-secret",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"x-service-name", "order",
				"x-service-secret", "correct-secret",
				"x-service-secret", "another-secret",
			)),
			want: codes.Unauthenticated,
		},
		{
			name:           "server authentication not configured",
			expectedName:   "order",
			expectedSecret: "",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"x-service-name", "order",
				"x-service-secret", "correct-secret",
			)),
			want: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewServiceAuthInterceptor(tt.expectedName, tt.expectedSecret)
			called := false

			_, err := interceptor(
				tt.ctx,
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/cart.v1.CartService/CheckoutCartForOrder"},
				func(context.Context, any) (any, error) {
					called = true
					return "ok", nil
				},
			)

			if status.Code(err) != tt.want {
				t.Fatalf("expected %s, got %v", tt.want, err)
			}
			if called != (tt.want == codes.OK) {
				t.Fatalf("handler called=%v, expected %v", called, tt.want == codes.OK)
			}
		})
	}
}
