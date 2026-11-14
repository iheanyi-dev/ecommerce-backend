package grpcserver

import (
	"context"
	"errors"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/proto/promotion/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	promotionv1.UnimplementedPromotionServiceServer

	create        *use_cases.CreatePromotionUseCase
	getByID       *use_cases.GetPromotionByIDUseCase
	getForUser    *use_cases.GetPromosForUserUseCase
	getOneForUser *use_cases.GetPromoForUserUseCase
	getByStore    *use_cases.GetPromosByStoreUseCase
	activate      *use_cases.ActivatePromotionUseCase
	cancel        *use_cases.CancelPromotionUseCase
	expire        *use_cases.ExpirePromotionUseCase
	renew         *use_cases.RenewPromotionUseCase
	delete        *use_cases.DeletePromotionUseCase
}

func NewServer(
	create *use_cases.CreatePromotionUseCase,
	getByID *use_cases.GetPromotionByIDUseCase,
	getForUser *use_cases.GetPromosForUserUseCase,
	getOneForUser *use_cases.GetPromoForUserUseCase,
	getByStore *use_cases.GetPromosByStoreUseCase,
	activate *use_cases.ActivatePromotionUseCase,
	cancel *use_cases.CancelPromotionUseCase,
	expire *use_cases.ExpirePromotionUseCase,
	renew *use_cases.RenewPromotionUseCase,
	deletePromotion *use_cases.DeletePromotionUseCase,
) *Server {
	return &Server{
		create:        create,
		getByID:       getByID,
		getForUser:    getForUser,
		getOneForUser: getOneForUser,
		getByStore:    getByStore,
		activate:      activate,
		cancel:        cancel,
		expire:        expire,
		renew:         renew,
		delete:        deletePromotion,
	}
}

func (s *Server) CreatePromotion(
	ctx context.Context,
	req *promotionv1.CreatePromotionRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	startsAt, err := parseTimestamp(req.GetStartsAt())
	if err != nil {
		return nil, err
	}
	endsAt, err := parseTimestamp(req.GetEndsAt())
	if err != nil {
		return nil, err
	}

	var fixedAmount *dtos.MoneyDTO
	if req.GetFixedAmount() != nil {
		fixedAmount = &dtos.MoneyDTO{
			Amount:   req.GetFixedAmount().GetAmount(),
			Currency: req.GetFixedAmount().GetCurrency(),
		}
	}

	result, err := s.create.Execute(ctx, dtos.CreatePromotionInput{
		StoreID:               req.GetStoreId(),
		ProductID:             req.ProductId,
		CustomerID:            req.CustomerId,
		PercentageBasisPoints: req.PercentageBasisPoints,
		FixedAmount:           fixedAmount,
		StartsAt:              startsAt,
		EndsAt:                endsAt,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) GetPromotionByID(
	ctx context.Context,
	req *promotionv1.GetPromotionByIDRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.getByID.Execute(ctx, dtos.GetPromotionByIDInput{
		PromotionID: req.GetPromotionId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) GetPromosForUser(
	ctx context.Context,
	req *promotionv1.GetPromosForUserRequest,
) (*promotionv1.ListPromotionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	results, err := s.getForUser.Execute(ctx, dtos.GetPromosForUserInput{
		CustomerID: req.GetCustomerId(),
		Limit:      int(req.GetLimit()),
		Offset:     int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return listPromotionsResponse(results), nil
}

func (s *Server) GetPromoForUser(
	ctx context.Context,
	req *promotionv1.GetPromoForUserRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.getOneForUser.Execute(ctx, dtos.GetPromoForUserInput{
		StoreID:    req.GetStoreId(),
		ProductID:  req.GetProductId(),
		CustomerID: req.GetCustomerId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) GetPromosByStore(
	ctx context.Context,
	req *promotionv1.GetPromosByStoreRequest,
) (*promotionv1.ListPromotionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	results, err := s.getByStore.Execute(ctx, dtos.GetPromosByStoreInput{
		StoreID: req.GetStoreId(),
		Limit:   int(req.GetLimit()),
		Offset:  int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return listPromotionsResponse(results), nil
}

func (s *Server) ActivatePromotion(
	ctx context.Context,
	req *promotionv1.PromotionLifecycleRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.activate.Execute(ctx, dtos.ActivatePromotionInput{
		PromotionID: req.GetPromotionId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) CancelPromotion(
	ctx context.Context,
	req *promotionv1.PromotionLifecycleRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.cancel.Execute(ctx, dtos.CancelPromotionInput{
		PromotionID: req.GetPromotionId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) ExpirePromotion(
	ctx context.Context,
	req *promotionv1.PromotionLifecycleRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.expire.Execute(ctx, dtos.ExpirePromotionInput{
		PromotionID: req.GetPromotionId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) RenewPromotion(
	ctx context.Context,
	req *promotionv1.RenewPromotionRequest,
) (*promotionv1.PromotionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	startsAt, err := parseTimestamp(req.GetStartsAt())
	if err != nil {
		return nil, err
	}
	endsAt, err := parseTimestamp(req.GetEndsAt())
	if err != nil {
		return nil, err
	}

	result, err := s.renew.Execute(ctx, dtos.RenewPromotionInput{
		PromotionID: req.GetPromotionId(),
		StartsAt:    startsAt,
		EndsAt:      endsAt,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return promotionResponse(result), nil
}

func (s *Server) DeletePromotion(
	ctx context.Context,
	req *promotionv1.PromotionLifecycleRequest,
) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	err := s.delete.Execute(ctx, dtos.DeletePromotionInput{
		PromotionID: req.GetPromotionId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &emptypb.Empty{}, nil
}

func parseTimestamp(value *timestamppb.Timestamp) (time.Time, error) {
	if value == nil {
		return time.Time{}, status.Error(codes.InvalidArgument, "timestamp is required")
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}, status.Error(codes.InvalidArgument, "timestamp is invalid")
	}
	return value.AsTime(), nil
}

func promotionResponse(dto dtos.PromotionDTO) *promotionv1.PromotionResponse {
	result := &promotionv1.Promotion{
		Id:        dto.ID,
		StoreId:   dto.StoreID,
		StartsAt:  timestamppb.New(dto.StartsAt),
		EndsAt:    timestamppb.New(dto.EndsAt),
		Status:    string(dto.Status),
		CreatedAt: timestamppb.New(dto.CreatedAt),
		UpdatedAt: timestamppb.New(dto.UpdatedAt),
	}

	if dto.ProductID != nil {
		result.ProductId = dto.ProductID
	}
	if dto.CustomerID != nil {
		result.CustomerId = dto.CustomerID
	}
	if dto.PercentageBasisPoints != nil {
		result.PercentageBasisPoints = dto.PercentageBasisPoints
	}
	if dto.FixedAmount != nil {
		result.FixedAmount = &promotionv1.Money{
			Amount:   dto.FixedAmount.Amount,
			Currency: dto.FixedAmount.Currency,
		}
	}

	return &promotionv1.PromotionResponse{Promotion: result}
}

func listPromotionsResponse(dtos []dtos.PromotionDTO) *promotionv1.ListPromotionsResponse {
	result := &promotionv1.ListPromotionsResponse{
		Promotions: make([]*promotionv1.Promotion, 0, len(dtos)),
	}
	for _, dto := range dtos {
		result.Promotions = append(result.Promotions, promotionResponse(dto).GetPromotion())
	}
	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, apperrors.ErrPromotionNotFound):
		return status.Error(codes.NotFound, "promotion not found")

	case errors.Is(err, apperrors.ErrPromotionConflict):
		return status.Error(codes.AlreadyExists, "promotion conflicts with an existing promotion")

	case errors.Is(err, apperrors.ErrInvalidPromotionQuery),
		errors.Is(err, domainerrors.ErrInvalidPromotion),
		errors.Is(err, valueobjects.ErrInvalidMoney),
		errors.Is(err, valueobjects.ErrInvalidDiscount):
		return status.Error(codes.InvalidArgument, "invalid promotion request")

	case errors.Is(err, apperrors.ErrPromotionNotDeletable),
		errors.Is(err, domainerrors.ErrPromotionAlreadyPaused),
		errors.Is(err, domainerrors.ErrPromotionNotPaused),
		errors.Is(err, domainerrors.ErrPromotionExpired),
		errors.Is(err, domainerrors.ErrPromotionCancelled),
		errors.Is(err, domainerrors.ErrPromotionAlreadyActive),
		errors.Is(err, domainerrors.ErrPromotionNotActive),
		errors.Is(err, domainerrors.ErrPromotionAlreadyExpired),
		errors.Is(err, domainerrors.ErrPromotionNotExpired),
		errors.Is(err, domainerrors.ErrPromotionPaused):
		return status.Error(codes.FailedPrecondition, "promotion lifecycle transition is not allowed")

	default:
		return status.Error(codes.Internal, "internal promotion service error")
	}
}
