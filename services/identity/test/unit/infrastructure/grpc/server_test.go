package grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	identitygrpc "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/grpc"
)

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

type mockPromoteUserToVendorService struct {
	execute func(context.Context, string) error
}

func (m *mockPromoteUserToVendorService) Execute(
	ctx context.Context,
	userID string,
) error {
	return m.execute(ctx, userID)
}

var _ ports.PromoteUserToVendorService = (*mockPromoteUserToVendorService)(nil)

func TestServer_GetUser_Success(t *testing.T) {
	userID := uuid.New()

	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(ctx context.Context, id string) (*dto.GetUserResult, error) {
				if id != userID.String() {
					t.Fatalf("unexpected user ID: %s", id)
				}

				return &dto.GetUserResult{
					ID:        userID.String(),
					Email:     "vendor@example.com",
					FullName:  "Vendor User",
					Role:      "vendor",
					Status:    "active",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return nil
			},
		},
	)

	response, err := server.GetUser(
		context.Background(),
		&identityv1.GetUserRequest{
			UserId: userID.String(),
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.GetUser() == nil {
		t.Fatal("expected user response")
	}

	if response.GetUser().GetId() != userID.String() {
		t.Fatalf("expected user ID %s, got %s", userID, response.GetUser().GetId())
	}

	if response.GetUser().GetEmail() != "vendor@example.com" {
		t.Fatalf("unexpected email: %s", response.GetUser().GetEmail())
	}
}

func TestServer_GetUser_Validation(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				t.Fatal("use case should not be called")
				return nil, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return nil
			},
		},
	)

	tests := []struct {
		name string
		req  *identityv1.GetUserRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing user ID",
			req:  &identityv1.GetUserRequest{},
		},
		{
			name: "invalid user ID",
			req:  &identityv1.GetUserRequest{UserId: "invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.GetUser(context.Background(), tt.req)

			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf(
					"expected InvalidArgument, got %s",
					status.Code(err),
				)
			}
		})
	}
}

func TestServer_GetUser_NotFound(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, application_errors.ErrUserNotFound
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return nil
			},
		},
	)

	_, err := server.GetUser(
		context.Background(),
		&identityv1.GetUserRequest{
			UserId: uuid.NewString(),
		},
	)

	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %s", status.Code(err))
	}
}

func TestServer_GetUser_InternalError(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, errors.New("database failure")
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return nil
			},
		},
	)

	_, err := server.GetUser(
		context.Background(),
		&identityv1.GetUserRequest{
			UserId: uuid.NewString(),
		},
	)

	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %s", status.Code(err))
	}
}

func TestServer_PromoteToVendor_Success(t *testing.T) {
	userID := uuid.New()
	called := false

	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(ctx context.Context, id string) error {
				called = true

				if id != userID.String() {
					t.Fatalf("unexpected user ID: %s", id)
				}

				return nil
			},
		},
	)

	response, err := server.PromoteToVendor(
		context.Background(),
		&identityv1.PromoteToVendorRequest{
			UserId: userID.String(),
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("expected response")
	}

	if !called {
		t.Fatal("expected promotion use case to be called")
	}
}

func TestServer_PromoteToVendor_Validation(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				t.Fatal("use case should not be called")
				return nil
			},
		},
	)

	tests := []struct {
		name string
		req  *identityv1.PromoteToVendorRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing user ID",
			req:  &identityv1.PromoteToVendorRequest{},
		},
		{
			name: "invalid user ID",
			req:  &identityv1.PromoteToVendorRequest{UserId: "invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.PromoteToVendor(
				context.Background(),
				tt.req,
			)

			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf(
					"expected InvalidArgument, got %s",
					status.Code(err),
				)
			}
		})
	}
}

func TestServer_PromoteToVendor_NotFound(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return application_errors.ErrUserNotFound
			},
		},
	)

	_, err := server.PromoteToVendor(
		context.Background(),
		&identityv1.PromoteToVendorRequest{
			UserId: uuid.NewString(),
		},
	)

	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %s", status.Code(err))
	}
}

func TestServer_PromoteToVendor_FailedPrecondition(t *testing.T) {
	server := identitygrpc.NewServer(
		&mockGetUserService{
			execute: func(context.Context, string) (*dto.GetUserResult, error) {
				return nil, nil
			},
		},
		&mockPromoteUserToVendorService{
			execute: func(context.Context, string) error {
				return errors.New("already vendor")
			},
		},
	)

	_, err := server.PromoteToVendor(
		context.Background(),
		&identityv1.PromoteToVendorRequest{
			UserId: uuid.NewString(),
		},
	)

	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s", status.Code(err))
	}
}
