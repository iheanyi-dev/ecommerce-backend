package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

func NewClient(lc fx.Lifecycle, cfg *config.Config) (ports.OrderClient, error) {
	if cfg.OrderServiceAddr == "" || cfg.OrderServiceName == "" || cfg.OrderServiceSecret == "" {
		return nil, fmt.Errorf("ORDER_SERVICE_ADDR, ORDER_SERVICE_NAME, and ORDER_SERVICE_SECRET are required")
	}
	conn, err := grpc.NewClient(cfg.OrderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create Order Service gRPC client: %w", err)
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return conn.Close() }})
	return &client{
		rpc:           orderv1.NewOrderServiceClient(conn),
		serviceName:   cfg.OrderServiceName,
		serviceSecret: cfg.OrderServiceSecret,
	}, nil
}

type client struct {
	rpc           orderv1.OrderServiceClient
	serviceName   string
	serviceSecret string
}

func (c *client) authenticated(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)
}

func (c *client) ListStoreOrders(ctx context.Context, storeID uuid.UUID, page, pageSize int) (*dto.ListStoreOrdersOutput, error) {
	if storeID == uuid.Nil || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, fmt.Errorf("list Store orders: invalid input")
	}
	res, err := c.rpc.ListStoreOrders(c.authenticated(ctx), &orderv1.ListStoreOrdersRequest{
		StoreId: storeID.String(), Page: int32(page), PageSize: int32(pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("list Store orders: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf("list Store orders: empty response")
	}
	out := &dto.ListStoreOrdersOutput{
		Page: int(res.GetPage()), PageSize: int(res.GetPageSize()),
		TotalItems: res.GetTotalItems(), TotalPages: res.GetTotalPages(),
		Orders: make([]dto.OrderOutput, 0, len(res.GetOrders())),
	}
	for _, item := range res.GetOrders() {
		converted, err := convertOrder(item)
		if err != nil {
			return nil, err
		}
		out.Orders = append(out.Orders, *converted)
	}
	return out, nil
}

func (c *client) GetStoreOrder(ctx context.Context, storeID, orderID uuid.UUID) (*dto.OrderOutput, error) {
	if storeID == uuid.Nil || orderID == uuid.Nil {
		return nil, fmt.Errorf("get Store order: IDs are required")
	}
	res, err := c.rpc.GetStoreOrder(c.authenticated(ctx), &orderv1.GetStoreOrderRequest{
		StoreId: storeID.String(), OrderId: orderID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("get Store order: %w", err)
	}
	if res == nil || res.GetOrder() == nil {
		return nil, fmt.Errorf("get Store order: empty response")
	}
	return convertOrder(res.GetOrder())
}

func (c *client) AddShipping(ctx context.Context, storeID, orderID uuid.UUID, amount int64) error {
	if storeID == uuid.Nil || orderID == uuid.Nil || amount < 0 {
		return fmt.Errorf("add shipping: invalid input")
	}
	res, err := c.rpc.AddShipping(c.authenticated(ctx), &orderv1.AddShippingRequest{
		StoreId: storeID.String(), OrderId: orderID.String(), ShippingAmount: amount,
	})
	if err != nil {
		return fmt.Errorf("add shipping: %w", err)
	}
	if res == nil || !res.GetSuccess() {
		return fmt.Errorf("add shipping: operation was not confirmed")
	}
	return nil
}

func (c *client) MarkSent(ctx context.Context, storeID, orderID uuid.UUID) error {
	if storeID == uuid.Nil || orderID == uuid.Nil {
		return fmt.Errorf("mark order sent: IDs are required")
	}
	res, err := c.rpc.MarkSent(c.authenticated(ctx), &orderv1.MarkSentRequest{StoreId: storeID.String(), OrderId: orderID.String()})
	if err != nil {
		return fmt.Errorf("mark order sent: %w", err)
	}
	if res == nil || !res.GetSuccess() {
		return fmt.Errorf("mark order sent: operation was not confirmed")
	}
	return nil
}

func (c *client) CancelStoreOrder(ctx context.Context, storeID, orderID uuid.UUID) error {
	if storeID == uuid.Nil || orderID == uuid.Nil {
		return fmt.Errorf("cancel Store order: IDs are required")
	}
	res, err := c.rpc.CancelStoreOrder(c.authenticated(ctx), &orderv1.CancelStoreOrderRequest{StoreId: storeID.String(), OrderId: orderID.String()})
	if err != nil {
		return fmt.Errorf("cancel Store order: %w", err)
	}
	if res == nil || !res.GetSuccess() {
		return fmt.Errorf("cancel Store order: operation was not confirmed")
	}
	return nil
}

func convertOrder(in *orderv1.Order) (*dto.OrderOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("convert Order response: missing order")
	}
	parse := func(field, value string) (uuid.UUID, error) {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return uuid.Nil, fmt.Errorf("convert Order response: invalid %s", field)
		}
		return id, nil
	}
	id, err := parse("order ID", in.GetId())
	if err != nil {
		return nil, err
	}
	storeID, err := parse("store ID", in.GetStoreId())
	if err != nil {
		return nil, err
	}
	customerID, err := parse("customer ID", in.GetCustomerId())
	if err != nil {
		return nil, err
	}
	cartID, err := parse("cart ID", in.GetCartId())
	if err != nil {
		return nil, err
	}
	created, err := timestamp(in.GetCreatedAt())
	if err != nil {
		return nil, fmt.Errorf("convert Order response: created_at: %w", err)
	}
	updated, err := timestamp(in.GetUpdatedAt())
	if err != nil {
		return nil, fmt.Errorf("convert Order response: updated_at: %w", err)
	}
	out := &dto.OrderOutput{
		ID: id, CartID: cartID, StoreID: storeID, CustomerID: customerID,
		DeliveryLocation: in.GetDeliveryLocation(), PriceProduct: in.GetPriceProduct(),
		PriceShipping: in.GetPriceShipping(), Currency: in.GetCurrency(), State: in.GetState(),
		CreatedAt: created, UpdatedAt: updated, Products: make([]dto.OrderProductOutput, 0, len(in.GetProducts())),
	}
	for _, p := range in.GetProducts() {
		if p == nil {
			return nil, fmt.Errorf("convert Order response: missing product")
		}
		pid, err := parse("product ID", p.GetProductId())
		if err != nil {
			return nil, err
		}
		lineID, err := parse("order product ID", p.GetId())
		if err != nil {
			return nil, err
		}
		lineOrderID, err := parse("product order ID", p.GetOrderId())
		if err != nil {
			return nil, err
		}
		if lineOrderID != id || p.GetQuantity() <= 0 || p.GetPriceMinor() < 0 {
			return nil, fmt.Errorf("convert Order response: invalid product line")
		}
		out.Products = append(out.Products, dto.OrderProductOutput{
			ID: lineID, OrderID: lineOrderID, ProductID: pid, Name: p.GetName(),
			Quantity: p.GetQuantity(), PriceMinor: p.GetPriceMinor(),
		})
	}
	return out, nil
}

func timestamp(in *timestamppb.Timestamp) (time.Time, error) {
	if in == nil || !in.IsValid() {
		return time.Time{}, fmt.Errorf("missing or invalid timestamp")
	}
	return in.AsTime(), nil
}
