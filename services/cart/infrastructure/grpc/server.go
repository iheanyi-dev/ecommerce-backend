// Package grpc implements Cart's trusted internal gRPC API.
package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	cartv1 "github.com/iheanyi-dev/ecommerce-backend/proto/cart/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checkoutForOrder defines the application capability exposed to Order.
type checkoutForOrder interface {
	Execute(
		ctx context.Context,
		customerID, cartID uuid.UUID,
	) (*use_cases.CheckoutCartOutput, error)
}

// Server implements the internal Cart gRPC service.
type Server struct {
	cartv1.UnimplementedCartServiceServer
	checkout checkoutForOrder
}

// NewServer constructs the Cart gRPC server adapter.
func NewServer(checkout *use_cases.CheckoutCartForOrder) *Server {
	return &Server{checkout: checkout}
}

// CheckoutCartForOrder reconciles the specified cart and returns the
// authoritative checkout snapshot required by Order.
func (s *Server) CheckoutCartForOrder(
	ctx context.Context,
	req *cartv1.CheckoutCartForOrderRequest,
) (*cartv1.CheckoutCartForOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil || customerID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "customer_id must be a valid UUID")
	}

	cartID, err := uuid.Parse(req.GetCartId())
	if err != nil || cartID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "cart_id must be a valid UUID")
	}

	if s.checkout == nil {
		return nil, status.Error(codes.Internal, "Cart checkout is unavailable")
	}

	result, err := s.checkout.Execute(ctx, customerID, cartID)
	if err != nil {
		switch {
		case errors.Is(err, domainerrors.ErrInvalidCustomerID),
			errors.Is(err, domainerrors.ErrInvalidCartID):
			return nil, status.Error(codes.InvalidArgument, "invalid checkout identity")
		case errors.Is(err, domainerrors.ErrCartNotFound),
			errors.Is(err, domainerrors.ErrCartOwnershipMismatch):
			return nil, status.Error(codes.NotFound, "cart not found for customer")
		default:
			return nil, status.Error(codes.Internal, "cart checkout failed")
		}
	}
	if result == nil {
		return nil, status.Error(codes.Internal, "cart checkout returned no result")
	}

	response := &cartv1.CheckoutCartForOrderResponse{
		CartId:            result.CartID.String(),
		Stores:            make([]*cartv1.CheckoutStore, 0, len(result.Stores)),
		RemovedProductIds: make([]string, 0, len(result.RemovedProductIDs)),
	}

	for _, store := range result.Stores {
		storeResponse := &cartv1.CheckoutStore{
			StoreId: store.StoreID.String(),
			Items:   make([]*cartv1.CheckoutItem, 0, len(store.Items)),
		}

		for _, item := range store.Items {
			storeResponse.Items = append(storeResponse.Items, &cartv1.CheckoutItem{
				ProductId:  item.Product.ID.String(),
				Name:       item.Product.Name,
				Quantity:   item.CartQuantity,
				PriceMinor: item.Product.PriceMinor,
				Currency:   item.Product.Currency,
			})
		}
		response.Stores = append(response.Stores, storeResponse)
	}

	for _, productID := range result.RemovedProductIDs {
		response.RemovedProductIds = append(response.RemovedProductIds, productID.String())
	}

	return response, nil
}
