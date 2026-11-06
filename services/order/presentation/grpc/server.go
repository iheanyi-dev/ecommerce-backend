package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ListStoreOrdersUseCase interface {
	Execute(context.Context, use_cases.ListOrdersInput) (*use_cases.OrderPage, error)
}

type GetStoreOrderUseCase interface {
	Execute(context.Context, use_cases.GetOrderInput) (*use_cases.OrderDetails, error)
}

type AddOrderShippingUseCase interface {
	Execute(context.Context, use_cases.AddShippingInput) error
}

type MarkOrderSentUseCase interface {
	Execute(context.Context, use_cases.MarkSentInput) error
}

type CancelStoreOrderUseCase interface {
	Execute(context.Context, use_cases.CancelOrderInput) error
}

type InitiateOrderPayoutUseCase interface {
	Execute(context.Context, use_cases.InitiatePayoutInput) (*use_cases.InitiatePayoutOutput, error)
}

// Server exposes authenticated, store-owner order operations.
type Server struct {
	orderv1.UnimplementedOrderServiceServer

	listOrders  ListStoreOrdersUseCase
	getOrder    GetStoreOrderUseCase
	addShipping AddOrderShippingUseCase
	markSent    MarkOrderSentUseCase
	cancelOrder CancelStoreOrderUseCase
	initPayout  InitiateOrderPayoutUseCase
}

func NewServer(
	listOrders ListStoreOrdersUseCase,
	getOrder GetStoreOrderUseCase,
	addShipping AddOrderShippingUseCase,
	markSent MarkOrderSentUseCase,
	cancelOrder CancelStoreOrderUseCase,
	initPayout InitiateOrderPayoutUseCase,
) *Server {
	return &Server{
		listOrders:  listOrders,
		getOrder:    getOrder,
		addShipping: addShipping,
		markSent:    markSent,
		cancelOrder: cancelOrder,
		initPayout:  initPayout,
	}
}

// ListOrders returns the authenticated store's paginated orders.
func (s *Server) ListOrders(
	ctx context.Context,
	req *orderv1.ListOrdersRequest,
) (*orderv1.ListOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	result, err := s.executeListOrders(ctx, storeID, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &orderv1.ListOrdersResponse{
		Orders:     result.Orders,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}, nil
}

// ListStoreOrders is retained for compatibility with existing Store clients.
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
	result, err := s.executeListOrders(ctx, storeID, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &orderv1.ListStoreOrdersResponse{
		Orders:     result.Orders,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *Server) executeListOrders(
	ctx context.Context,
	storeID uuid.UUID,
	requestedPage int32,
	requestedPageSize int32,
) (*listOrdersResult, error) {
	if requestedPage < 0 || requestedPageSize < 0 {
		return nil, status.Error(codes.InvalidArgument, "page and page_size cannot be negative")
	}
	if s.listOrders == nil {
		return nil, status.Error(codes.Internal, "order listing is unavailable")
	}

	page := int(requestedPage)
	if page == 0 {
		page = 1
	}

	result, err := s.listOrders.Execute(ctx, use_cases.ListOrdersInput{
		UserID:   storeID,
		StoreID:  storeID,
		Page:     page,
		PageSize: int(requestedPageSize),
	})
	if err != nil {
		return nil, mapError(err)
	}
	if result == nil {
		return nil, status.Error(codes.Internal, "order listing returned no result")
	}

	response := &listOrdersResult{
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

type listOrdersResult struct {
	Orders     []*orderv1.Order
	Page       int32
	PageSize   int32
	TotalItems int64
	TotalPages int64
}

// GetOrder retrieves an order within the supplied store scope.
func (s *Server) GetOrder(
	ctx context.Context,
	req *orderv1.GetOrderRequest,
) (*orderv1.GetOrderResponse, error) {
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
	result, err := s.executeGetOrder(ctx, orderID, storeID)
	if err != nil {
		return nil, err
	}
	return &orderv1.GetOrderResponse{
		Order: toProtoOrder(result.Order, result.Products),
	}, nil
}

// GetStoreOrder is retained for compatibility with existing Store clients.
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
	result, err := s.executeGetOrder(ctx, orderID, storeID)
	if err != nil {
		return nil, err
	}
	return &orderv1.GetStoreOrderResponse{
		Order: toProtoOrder(result.Order, result.Products),
	}, nil
}

func (s *Server) executeGetOrder(
	ctx context.Context,
	orderID uuid.UUID,
	storeID uuid.UUID,
) (*use_cases.OrderDetails, error) {
	if s.getOrder == nil {
		return nil, status.Error(codes.Internal, "order retrieval is unavailable")
	}
	result, err := s.getOrder.Execute(ctx, use_cases.GetOrderInput{
		OrderID: orderID,
		UserID:  storeID,
		StoreID: storeID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	if result == nil || result.Order == nil {
		return nil, status.Error(codes.Internal, "order retrieval returned an invalid result")
	}
	return result, nil
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
	if err := s.addShipping.Execute(ctx, use_cases.AddShippingInput{
		OrderID:        orderID,
		StoreID:        storeID,
		ShippingAmount: req.GetShippingAmount(),
	}); err != nil {
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
		OrderID: orderID,
		StoreID: storeID,
	}); err != nil {
		return nil, mapError(err)
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

// InitiatePayout initiates a payout for an order belonging to the supplied store.
func (s *Server) InitiatePayout(
	ctx context.Context,
	req *orderv1.InitiatePayoutRequest,
) (*orderv1.InitiatePayoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	orderID, err := parseID(req.GetOrderId(), "order_id")
	if err != nil {
		return nil, err
	}
	storeID, err := parseID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, err
	}
	if s.initPayout == nil {
		return nil, status.Error(codes.Internal, "payout operation is unavailable")
	}

	result, err := s.initPayout.Execute(ctx, use_cases.InitiatePayoutInput{
		OrderID: orderID,
		StoreID: storeID,
		Beneficiary: ports.PayoutBeneficiary{
			AccountName:   req.GetAccountName(),
			AccountNumber: req.GetAccountNumber(),
			BankCode:      req.GetBankCode(),
			BankName:      req.GetBankName(),
			CountryCode:   req.GetCountryCode(),
		},
	})
	if err != nil {
		return nil, mapError(err)
	}
	if result == nil {
		return nil, status.Error(codes.Internal, "payout operation returned no result")
	}

	return &orderv1.InitiatePayoutResponse{
		PayoutId:      result.PayoutID,
		AttemptId:     result.AttemptID,
		Reference:     result.Reference,
		Status:        result.Status,
		AttemptStatus: result.AttemptStatus,
		Provider:      result.Provider,
		AmountMinor:   result.AmountMinor,
		Currency:      result.Currency,
	}, nil
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

	// The store scope comes from the authenticated internal Store service boundary.
	if err := s.cancelOrder.Execute(ctx, use_cases.CancelOrderInput{
		OrderID: orderID,
		ActorID: storeID,
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
			Id:         product.ID.String(),
			OrderId:    product.OrderID.String(),
			ProductId:  product.ProductID.String(),
			Name:       product.Name,
			Quantity:   product.Qty,
			PriceMinor: product.Price,
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
		return status.Error(codes.Internal, "order operation failed")
	}
}

var _ orderv1.OrderServiceServer = (*Server)(nil)
