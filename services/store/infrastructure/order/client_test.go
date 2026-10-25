package order

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const orderBufferSize = 1024 * 1024

type orderServerMock struct {
	orderv1.UnimplementedOrderServiceServer
	t             *testing.T
	fail          bool
	storeID       string
	orderID       string
	authenticated bool
}

func (s *orderServerMock) checkAuth(ctx context.Context) {
	s.t.Helper()
	md, ok := metadata.FromIncomingContext(ctx)
	require.True(s.t, ok)
	require.Equal(s.t, []string{"store"}, md.Get("x-service-name"))
	require.Equal(s.t, []string{"store-secret"}, md.Get("x-service-secret"))
	s.authenticated = true
}

func (s *orderServerMock) ListStoreOrders(ctx context.Context, req *orderv1.ListStoreOrdersRequest) (*orderv1.ListStoreOrdersResponse, error) {
	s.checkAuth(ctx)
	s.storeID = req.GetStoreId()
	if s.fail {
		return nil, status.Error(codes.Unavailable, "Order service unavailable")
	}
	require.Equal(s.t, int32(2), req.GetPage())
	require.Equal(s.t, int32(10), req.GetPageSize())

	now := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	return &orderv1.ListStoreOrdersResponse{
		Page: 2, PageSize: 10, TotalItems: 21, TotalPages: 3,
		Orders: []*orderv1.Order{{
			Id: s.orderID, StoreId: s.storeID,
			CustomerId: uuid.NewString(), CartId: uuid.NewString(),
			DeliveryLocation: "Lagos", PriceProduct: 5000,
			PriceShipping: 300, Currency: "NGN", State: "paid",
			CreatedAt: now, UpdatedAt: now,
			Products: []*orderv1.OrderProduct{{
				Id: uuid.NewString(), OrderId: s.orderID,
				ProductId: uuid.NewString(), Name: "Product",
				Quantity: 2, PriceMinor: 2500,
			}},
		}},
	}, nil
}

func (s *orderServerMock) GetStoreOrder(ctx context.Context, req *orderv1.GetStoreOrderRequest) (*orderv1.GetStoreOrderResponse, error) {
	s.checkAuth(ctx)
	if s.fail {
		return nil, status.Error(codes.Unavailable, "Order service unavailable")
	}
	return &orderv1.GetStoreOrderResponse{}, nil
}

func (s *orderServerMock) AddShipping(ctx context.Context, req *orderv1.AddShippingRequest) (*orderv1.OrderActionResponse, error) {
	s.checkAuth(ctx)
	if s.fail {
		return nil, status.Error(codes.Unavailable, "Order service unavailable")
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func (s *orderServerMock) MarkSent(ctx context.Context, req *orderv1.MarkSentRequest) (*orderv1.OrderActionResponse, error) {
	s.checkAuth(ctx)
	if s.fail {
		return nil, status.Error(codes.Unavailable, "Order service unavailable")
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func (s *orderServerMock) CancelStoreOrder(ctx context.Context, req *orderv1.CancelStoreOrderRequest) (*orderv1.OrderActionResponse, error) {
	s.checkAuth(ctx)
	if s.fail {
		return nil, status.Error(codes.Unavailable, "Order service unavailable")
	}
	return &orderv1.OrderActionResponse{Success: true}, nil
}

func newTestClient(t *testing.T, fail bool) (*client, *orderServerMock) {
	t.Helper()
	listener := bufconn.Listen(orderBufferSize)
	server := grpc.NewServer()
	mock := &orderServerMock{
		t: t, fail: fail,
		storeID: uuid.NewString(),
		orderID: uuid.NewString(),
	}
	orderv1.RegisterOrderServiceServer(server, mock)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &client{
		rpc:           orderv1.NewOrderServiceClient(conn),
		serviceName:   "store",
		serviceSecret: "store-secret",
	}, mock
}

func TestListStoreOrdersMapsResponseAndAuthenticates(t *testing.T) {
	c, server := newTestClient(t, false)
	storeID, err := uuid.Parse(server.storeID)
	require.NoError(t, err)

	out, err := c.ListStoreOrders(context.Background(), storeID, 2, 10)

	require.NoError(t, err)
	require.True(t, server.authenticated)
	require.Equal(t, server.storeID, server.storeID)
	require.Equal(t, 2, out.Page)
	require.Equal(t, 10, out.PageSize)
	require.EqualValues(t, 21, out.TotalItems)
	require.EqualValues(t, 3, out.TotalPages)
	require.Len(t, out.Orders, 1)

	order := out.Orders[0]
	require.Equal(t, server.orderID, order.ID.String())
	require.Equal(t, storeID, order.StoreID)
	require.Equal(t, "NGN", order.Currency)
	require.EqualValues(t, 5000, order.PriceProduct)
	require.EqualValues(t, 300, order.PriceShipping)
	require.Len(t, order.Products, 1)
	require.Equal(t, "Product", order.Products[0].Name)
	require.EqualValues(t, 2, order.Products[0].Quantity)
	require.EqualValues(t, 2500, order.Products[0].PriceMinor)
	require.False(t, order.CreatedAt.IsZero())
}

func TestListStoreOrdersRejectsInvalidInput(t *testing.T) {
	c, _ := newTestClient(t, false)

	_, err := c.ListStoreOrders(context.Background(), uuid.Nil, 1, 10)
	require.Error(t, err)

	_, err = c.ListStoreOrders(context.Background(), uuid.New(), 0, 10)
	require.Error(t, err)

	_, err = c.ListStoreOrders(context.Background(), uuid.New(), 1, 101)
	require.Error(t, err)
}

func TestListStoreOrdersPropagatesRPCFailure(t *testing.T) {
	c, server := newTestClient(t, true)
	storeID, err := uuid.Parse(server.storeID)
	require.NoError(t, err)

	_, err = c.ListStoreOrders(context.Background(), storeID, 1, 10)

	require.Error(t, err)
	require.ErrorContains(t, err, "Order service unavailable")
	require.True(t, server.authenticated)
}

func TestActionMethodsRequireConfirmedSuccess(t *testing.T) {
	c, _ := newTestClient(t, false)
	storeID, orderID := uuid.New(), uuid.New()

	require.NoError(t, c.AddShipping(context.Background(), storeID, orderID, 300))
	require.NoError(t, c.MarkSent(context.Background(), storeID, orderID))
	require.NoError(t, c.CancelStoreOrder(context.Background(), storeID, orderID))

	require.Error(t, c.AddShipping(context.Background(), uuid.Nil, orderID, 300))
	require.Error(t, c.MarkSent(context.Background(), storeID, uuid.Nil))
	require.Error(t, c.CancelStoreOrder(context.Background(), storeID, uuid.Nil))
}

var _ orderv1.OrderServiceServer = (*orderServerMock)(nil)
var _ = dto.OrderOutput{}
