package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer

	getUser         ports.GetUserService
	promoteToVendor ports.PromoteUserToVendorService
}

func NewServer(
	getUser ports.GetUserService,
	promoteToVendor ports.PromoteUserToVendorService,
) *Server {
	return &Server{
		getUser:         getUser,
		promoteToVendor: promoteToVendor,
	}
}

func (s *Server) GetUser(
	ctx context.Context,
	req *identityv1.GetUserRequest,
) (*identityv1.GetUserResponse, error) {
	if req == nil || req.GetUserId() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user_id is required",
		)
	}

	if _, err := uuid.Parse(req.GetUserId()); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"invalid user_id",
		)
	}

	result, err := s.getUser.Execute(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, application_errors.ErrUserNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"user not found",
			)
		}

		return nil, status.Error(
			codes.Internal,
			"failed to retrieve user",
		)
	}

	return &identityv1.GetUserResponse{
		User: &identityv1.User{
			Id:       result.ID,
			Email:    result.Email,
			FullName: result.FullName,
			Role:     result.Role,
			Status:   result.Status,
		},
	}, nil
}

func (s *Server) PromoteToVendor(
	ctx context.Context,
	req *identityv1.PromoteToVendorRequest,
) (*identityv1.PromoteToVendorResponse, error) {
	if req == nil || req.GetUserId() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user_id is required",
		)
	}

	if _, err := uuid.Parse(req.GetUserId()); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"invalid user_id",
		)
	}

	err := s.promoteToVendor.Execute(ctx, req.GetUserId())
	if err != nil {
		switch {
		case errors.Is(err, application_errors.ErrUserNotFound):
			return nil, status.Error(
				codes.NotFound,
				"user not found",
			)

		default:
			return nil, status.Error(
				codes.FailedPrecondition,
				"unable to promote user to vendor",
			)
		}
	}

	return &identityv1.PromoteToVendorResponse{}, nil
}

var _ identityv1.IdentityServiceServer = (*Server)(nil)
