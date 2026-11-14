package grpcserver_test

import (
	"context"
	"testing"

	grpcserver "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServiceAuthInterceptor(t *testing.T) {
	tests := []struct {
		name         string
		configured   bool
		serviceNames []string
		secrets      []string
		wantCode     codes.Code
		wantHandler  bool
	}{
		{
			name:         "valid credentials",
			configured:   true,
			serviceNames: []string{"store"},
			secrets:      []string{"shared-secret"},
			wantCode:     codes.OK,
			wantHandler:  true,
		},
		{
			name:       "missing service name",
			configured: true,
			secrets:    []string{"shared-secret"},
			wantCode:   codes.Unauthenticated,
		},
		{
			name:         "missing service secret",
			configured:   true,
			serviceNames: []string{"store"},
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "invalid service name",
			configured:   true,
			serviceNames: []string{"billing"},
			secrets:      []string{"shared-secret"},
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "invalid service secret",
			configured:   true,
			serviceNames: []string{"store"},
			secrets:      []string{"wrong-secret"},
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "duplicate service names",
			configured:   true,
			serviceNames: []string{"store", "store"},
			secrets:      []string{"shared-secret"},
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "duplicate service secrets",
			configured:   true,
			serviceNames: []string{"store"},
			secrets:      []string{"shared-secret", "shared-secret"},
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "missing configured service name",
			serviceNames: []string{"store"},
			secrets:      []string{"shared-secret"},
			wantCode:     codes.Internal,
		},
		{
			name:         "missing configured secret",
			configured:   false,
			serviceNames: []string{"store"},
			secrets:      []string{"shared-secret"},
			wantCode:     codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedName := ""
			expectedSecret := ""

			if tt.configured {
				expectedName = "store"
				expectedSecret = "shared-secret"
			}

			if tt.name == "missing configured service name" {
				expectedSecret = "shared-secret"
			}

			if tt.name == "missing configured secret" {
				expectedName = "store"
			}

			interceptor := grpcserver.NewServiceAuthInterceptor(
				expectedName,
				expectedSecret,
			)

			md := metadata.MD{}
			if len(tt.serviceNames) > 0 {
				md.Set("x-service-name", tt.serviceNames...)
			}
			if len(tt.secrets) > 0 {
				md.Set("x-service-secret", tt.secrets...)
			}

			ctx := metadata.NewIncomingContext(context.Background(), md)

			handlerCalled := false
			handler := func(context.Context, any) (any, error) {
				handlerCalled = true
				return "success", nil
			}

			_, err := interceptor(
				ctx,
				nil,
				&grpc.UnaryServerInfo{
					FullMethod: "/promotion.v1.PromotionService/GetPromotionByID",
				},
				handler,
			)

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("status.Code() = %v, want %v; error: %v", got, tt.wantCode, err)
			}

			if handlerCalled != tt.wantHandler {
				t.Fatalf("handlerCalled = %v, want %v", handlerCalled, tt.wantHandler)
			}
		})
	}
}
