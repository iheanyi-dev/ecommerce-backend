package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	storev1 "github.com/iheanyi-dev/ecommerce-backend/proto/store/v1"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements Store's internal service-to-service API.
type Server struct {
	storev1.UnimplementedStoreServiceServer

	stores        ports.StoreRepository
	productCounts ports.ProductCapacityRepository
	activeStores  ports.ActiveStoreLookupRepository
}

// NewServer constructs the internal Store gRPC adapter.
func NewServer(
	stores ports.StoreRepository,
	productCounts ports.ProductCapacityRepository,
	activeStores ports.ActiveStoreLookupRepository,
) *Server {
	return &Server{
		stores:        stores,
		productCounts: productCounts,
		activeStores:  activeStores,
	}
}

var _ storev1.StoreServiceServer = (*Server)(nil)

func (s *Server) GetStoreIDByOwnerID(
	ctx context.Context,
	request *storev1.GetStoreIDByOwnerIDRequest,
) (*storev1.GetStoreIDByOwnerIDResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	ownerID, err := parseID(request.GetOwnerId(), "owner_id")
	if err != nil {
		return nil, err
	}

	store, err := s.stores.FindByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &storev1.GetStoreIDByOwnerIDResponse{
		StoreId: store.ID().Value().String(),
	}, nil
}

func (s *Server) IncrementProductCount(
	ctx context.Context,
	request *storev1.IncrementProductCountRequest,
) (*storev1.IncrementProductCountResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	storeID, err := parseID(request.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.productCounts.IncrementProductCount(ctx, storeID); err != nil {
		return nil, mapGRPCError(err)
	}

	return &storev1.IncrementProductCountResponse{}, nil
}

func (s *Server) DecrementProductCount(
	ctx context.Context,
	request *storev1.DecrementProductCountRequest,
) (*storev1.DecrementProductCountResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	storeID, err := parseID(request.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.productCounts.DecrementProductCount(ctx, storeID); err != nil {
		return nil, mapGRPCError(err)
	}

	return &storev1.DecrementProductCountResponse{}, nil
}

func (s *Server) IsStoreActive(
	ctx context.Context,
	request *storev1.IsStoreActiveRequest,
) (*storev1.IsStoreActiveResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	storeID, err := parseID(request.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}

	store, err := s.stores.FindByID(ctx, storeID)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &storev1.IsStoreActiveResponse{
		Active: store.Status() == valueobjects.StatusActive,
	}, nil
}

func (s *Server) FilterActiveStoreIDs(
	ctx context.Context,
	request *storev1.FilterActiveStoreIDsRequest,
) (*storev1.FilterActiveStoreIDsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	storeIDs := make([]uuid.UUID, 0, len(request.GetStoreIds()))
	for _, rawID := range request.GetStoreIds() {
		id, err := parseID(rawID, "store_ids")
		if err != nil {
			return nil, err
		}
		storeIDs = append(storeIDs, id)
	}

	activeIDs, err := s.activeStores.FilterActiveStoreIDs(ctx, storeIDs)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	response := &storev1.FilterActiveStoreIDsResponse{
		ActiveStoreIds: make([]string, 0, len(activeIDs)),
	}
	for _, id := range activeIDs {
		response.ActiveStoreIds = append(response.ActiveStoreIds, id.String())
	}

	return response, nil
}

func parseID(raw, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "%s must be a valid non-zero UUID", field)
	}
	return id, nil
}

func mapGRPCError(err error) error {
	switch {
	case errors.Is(err, applicationerrors.ErrStoreNotFound):
		return status.Error(codes.NotFound, "store not found")
	case errors.Is(err, applicationerrors.ErrStoreProductCapacityExceeded),
		errors.Is(err, applicationerrors.ErrStorePlanProductLimitExceeded):
		return status.Error(codes.ResourceExhausted, "store product capacity exceeded")
	case errors.Is(err, applicationerrors.ErrStoreProductCountUnderflow):
		return status.Error(codes.FailedPrecondition, "store product count cannot be negative")
	default:
		return status.Error(codes.Internal, fmt.Sprintf("store operation failed: %v", err))
	}
}
