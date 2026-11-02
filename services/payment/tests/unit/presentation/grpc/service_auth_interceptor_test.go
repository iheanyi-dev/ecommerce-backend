package grpc_test

import (
	"context"
	"testing"

	grpcserver "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServiceAuthInterceptor(t *testing.T) {
	const (
		serviceName   = "order-service"
		serviceSecret = "test-secret"
	)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "success", nil
	}

	tests := []struct {
		name         string
		configName   string
		configSecret string
		metadata     metadata.MD
		wantCode     codes.Code
		wantHandler  bool
	}{
		{
			name:         "valid credentials",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata: metadata.Pairs(
				"x-service-name", serviceName,
				"x-service-secret", serviceSecret,
			),
			wantCode:    codes.OK,
			wantHandler: true,
		},
		{
			name:         "missing metadata",
			configName:   serviceName,
			configSecret: serviceSecret,
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "missing service name",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata:     metadata.Pairs("x-service-secret", serviceSecret),
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "missing service secret",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata:     metadata.Pairs("x-service-name", serviceName),
			wantCode:     codes.Unauthenticated,
		},
		{
			name:         "incorrect service name",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata: metadata.Pairs(
				"x-service-name", "unknown-service",
				"x-service-secret", serviceSecret,
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:         "incorrect service secret",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata: metadata.Pairs(
				"x-service-name", serviceName,
				"x-service-secret", "wrong-secret",
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:         "duplicate service name",
			configName:   serviceName,
			configSecret: serviceSecret,
			metadata: metadata.Pairs(
				"x-service-name", serviceName,
				"x-service-name", serviceName,
				"x-service-secret", serviceSecret,
			),
			wantCode: codes.Unauthenticated,
		},
		{
			name:         "empty configured credentials",
			configName:   "",
			configSecret: serviceSecret,
			metadata: metadata.Pairs(
				"x-service-name", serviceName,
				"x-service-secret", serviceSecret,
			),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			testInterceptor := grpcserver.NewServiceAuthInterceptor(
				tt.configName,
				tt.configSecret,
			)

			ctx := context.Background()
			if tt.metadata != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.metadata)
			}

			_, err := testInterceptor(
				ctx,
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/payment.v1.PaymentService/Test"},
				handler,
			)

			if tt.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else if status.Code(err) != tt.wantCode {
				t.Fatalf("expected code %s, got %s (error: %v)", tt.wantCode, status.Code(err), err)
			}

			if handlerCalled != tt.wantHandler {
				t.Fatalf("expected handler called=%t, got %t", tt.wantHandler, handlerCalled)
			}
		})
	}
}
