package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type listStoreOrders interface {
	Execute(context.Context, use_cases.ListOrdersInput) (*use_cases.OrderPage, error)
}

type getStoreOrder interface {
	Execute(context.Context, use_cases.GetOrderInput) (*use_cases.OrderDetails, error)
}

type addOrderShipping interface {
	Execute(context.Context, use_cases.AddShippingInput) error
}

type markOrderSent interface {
	Execute(context.Context, use_cases.MarkSentInput) error
}

type cancelStoreOrder interface {
	Execute(context.Context, use_cases.CancelOrderInput) error
}

// Server exposes authenticated, store-owner order operations.
type Server struct {
	orderv1.UnimplementedOrderServiceServer

	listOrders  listStoreOrders
	getOrder    getStoreOrder
	addShipping addOrderShipping
	markSent    markOrderSent
	cancelOrder cancelStoreOrder
}

func NewServer(
	listOrders *use_cases.ListOrders,
	getOrder *use_cases.GetOrder,
	addShipping *use_cases.AddShipping,
	markSent *use_cases.MarkSent,
	cancelOrder *use_cases.CancelOrder,
) *Server {
	return &Server{
		listOrders:  listOrders,
		getOrder:    getOrder,
		addShipping: addShipping,
		markSent:    markSent,
		cancelOrder: cancelOrder,
	}
}

func (s *Server) ListStoreOrders(
	ctx context.Context,
	req *orderv1.ListStoreOrdersRequest,
) (*orderv1.ListStoreOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	if req.GetPage() < 0 || req.GetPageSize() < 0 {
		return nil, status.Error(codes.InvalidArgument, "page and page_size cannot be negative")
	}
	if s.listOrders == nil {
		return nil, status.Error(codes.Internal, "order listing is unavailable")
	}

	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}

	result, err := s.listOrders.Execute(ctx, use_cases.ListOrdersInput{
		UserID: storeID, StoreID: storeID,
		Page: page, PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	if result == nil {
		return nil, status.Error(codes.Internal, "order listing returned no result")
	}

	response := &orderv1.ListStoreOrdersResponse{
		Orders:     make([]*orderv1.Order, 0, len(result.Items)),
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}
	for _, item := range result.Items {
		if item == nil || item.Order == nil {
			return nil, status.Error(codes.Internal, "order listing returned an invalid result")
		}
		response.Orders = append(response.Orders, toProtoOrder(item.Order, item.Products))
	}
	return response, nil
}

func (s *Server) GetStoreOrder(
	ctx context.Context,
	req *orderv1.GetStoreOrderRequest,
) (*orderv1.GetStoreOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	orderID, err := parseID(req.GetOrderId(), "order_id")
	if err != nil {
		return nil, err
	}
	if s.getOrder == nil {
		return nil, status.Error(codes.Internal, "order retrieval is unavailable")
	}

	result, err := s.getOrder.Execute(ctx, use_cases.GetOrderInput{
		OrderID: orderID, UserID: storeID, StoreID: storeID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	if result == nil || result.Order == nil {
		return nil, status.Error(codes.Internal, "order retrieval returned an invalid result")
	}

	return &orderv1.GetStoreOrderResponse{
		Order: toProtoOrder(result.Order, result.Products),
	}, nil
}

func (s *Server) AddShipping(
	ctx context.Context,
	req *orderv1.AddShippingRequest,
) (*orderv1.OrderActionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	orderID, err := parseID(req.GetOrderId(), "order_id")
	if err != nil {
		return nil, err
	}
	if req.GetShippingAmount() < 0 {
		return nil, status.Error(codes.InvalidArgument, "shipping_amount cannot be negative")
	}
	if s.addShipping == nil {
		return nil, status.Error(codes.Internal, "shipping operation is unavailable")
	}

	err = s.addShipping.Execute(ctx, use_cases.AddShippingInput{
		OrderID: orderID, StoreID: storeID,
		ShippingAmount: req.GetShippingAmount(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func (s *Server) MarkSent(
	ctx context.Context,
	req *orderv1.MarkSentRequest,
) (*orderv1.OrderActionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	orderID, err := parseID(req.GetOrderId(), "order_id")
	if err != nil {
		return nil, err
	}
	if s.markSent == nil {
		return nil, status.Error(codes.Internal, "dispatch operation is unavailable")
	}

	if err := s.markSent.Execute(ctx, use_cases.MarkSentInput{
		OrderID: orderID, StoreID: storeID,
	}); err != nil {
		return nil, mapError(err)
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func (s *Server) CancelStoreOrder(
	ctx context.Context,
	req *orderv1.CancelStoreOrderRequest,
) (*orderv1.OrderActionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	orderID, err := parseID(req.GetOrderId(), "order_id")
	if err != nil {
		return nil, err
	}
	if s.cancelOrder == nil {
		return nil, status.Error(codes.Internal, "order cancellation is unavailable")
	}

	// The caller cannot choose ActorID; it is always the supplied store scope.
	// The authenticated internal service boundary must ensure StoreID is trusted.
	if err := s.cancelOrder.Execute(ctx, use_cases.CancelOrderInput{
		OrderID: orderID, ActorID: storeID,
	}); err != nil {
		return nil, mapError(err)
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func parseID(value, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "%s must be a valid UUID", field)
	}
	return id, nil
}

func toProtoOrder(order *entities.Order, products []*entities.OrderProduct) *orderv1.Order {
	result := &orderv1.Order{
		Id:               order.ID.String(),
		CartId:           order.CartID.String(),
		StoreId:          order.StoreID.String(),
		CustomerId:       order.CustomerID.String(),
		DeliveryLocation: order.DeliveryLocation,
		PriceProduct:     order.PriceProduct,
		PriceShipping:    order.PriceShipping,
		Currency:         order.Currency,
		State:            string(order.State),
		Products:         make([]*orderv1.OrderProduct, 0, len(products)),
	}
	if !order.CreatedAt.IsZero() {
		result.CreatedAt = timestamppb.New(order.CreatedAt)
	}
	if !order.UpdatedAt.IsZero() {
		result.UpdatedAt = timestamppb.New(order.UpdatedAt)
	}
	for _, product := range products {
		if product == nil {
			continue
		}
		result.Products = append(result.Products, &orderv1.OrderProduct{
			Id: product.ID.String(), OrderId: product.OrderID.String(),
			ProductId: product.ProductID.String(), Name: product.Name,
			Quantity: product.Qty, PriceMinor: product.Price,
		})
	}
	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthorizedActor):
		return status.Error(codes.PermissionDenied, "actor is not authorized for this order")
	case errors.Is(err, domain.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, "order cannot transition from its current state")
	case errors.Is(err, domain.ErrInvalidShippingAmount),
		errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, "order operation contains invalid data")
	case strings.Contains(strings.ToLower(err.Error()), "not found"):
		return status.Error(codes.NotFound, "order not found")
	default:
		_ = fmt.Sprintf("%v", err) // Keep internal error details out of RPC responses.
		return status.Error(codes.Internal, "order operation failed")
	}
}

var _ orderv1.OrderServiceServer = (*Server)(nil)

// Keep timestamp conversion explicit and UTC-normalized at the transport boundary.
func normalizedTimestamp(value time.Time) *timestamppb.Timestamp {
	return timestamppb.New(value.UTC())
}
