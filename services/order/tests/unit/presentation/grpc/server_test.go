package grpc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	ordergrpc "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type listOrdersMock struct {
	input use_cases.ListOrdersInput
	page  *use_cases.OrderPage
	err   error
}

func (m *listOrdersMock) Execute(
	_ context.Context,
	input use_cases.ListOrdersInput,
) (*use_cases.OrderPage, error) {
	m.input = input
	return m.page, m.err
}

type getOrderMock struct {
	input  use_cases.GetOrderInput
	result *use_cases.OrderDetails
	err    error
}

func (m *getOrderMock) Execute(
	_ context.Context,
	input use_cases.GetOrderInput,
) (*use_cases.OrderDetails, error) {
	m.input = input
	return m.result, m.err
}

type addShippingMock struct {
	input use_cases.AddShippingInput
	err   error
}

func (m *addShippingMock) Execute(
	_ context.Context,
	input use_cases.AddShippingInput,
) error {
	m.input = input
	return m.err
}

type markSentMock struct {
	input use_cases.MarkSentInput
	err   error
}

func (m *markSentMock) Execute(
	_ context.Context,
	input use_cases.MarkSentInput,
) error {
	m.input = input
	return m.err
}

type cancelOrderMock struct {
	input use_cases.CancelOrderInput
	err   error
}

func (m *cancelOrderMock) Execute(
	_ context.Context,
	input use_cases.CancelOrderInput,
) error {
	m.input = input
	return m.err
}

func TestListStoreOrdersUsesStoreScopeAndDefaultPagination(t *testing.T) {
	storeID := uuid.New()
	mock := &listOrdersMock{
		page: &use_cases.OrderPage{
			Page:     1,
			PageSize: 20,
		},
	}
	server := ordergrpc.NewServer(mock, nil, nil, nil, nil, nil)

	response, err := server.ListStoreOrders(
		context.Background(),
		&orderv1.ListStoreOrdersRequest{StoreId: storeID.String()},
	)
	if err != nil {
		t.Fatalf("ListStoreOrders() error = %v", err)
	}

	if response.GetPage() != 1 || response.GetPageSize() != 20 {
		t.Fatalf(
			"unexpected pagination: page=%d page_size=%d",
			response.GetPage(),
			response.GetPageSize(),
		)
	}
	if mock.input.StoreID != storeID || mock.input.UserID != storeID {
		t.Fatalf("store scope was not propagated: %+v", mock.input)
	}
	if mock.input.Page != 1 {
		t.Fatalf("default page = %d, want 1", mock.input.Page)
	}
}

func TestListStoreOrdersRejectsInvalidStoreID(t *testing.T) {
	server := ordergrpc.NewServer(&listOrdersMock{}, nil, nil, nil, nil, nil)

	_, err := server.ListStoreOrders(
		context.Background(),
		&orderv1.ListStoreOrdersRequest{StoreId: "not-a-uuid"},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestListStoreOrdersRejectsNilResult(t *testing.T) {
	server := ordergrpc.NewServer(&listOrdersMock{}, nil, nil, nil, nil, nil)

	_, err := server.ListStoreOrders(
		context.Background(),
		&orderv1.ListStoreOrdersRequest{StoreId: uuid.NewString()},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestListStoreOrdersRejectsNegativePagination(t *testing.T) {
	server := ordergrpc.NewServer(&listOrdersMock{}, nil, nil, nil, nil, nil)

	_, err := server.ListStoreOrders(
		context.Background(),
		&orderv1.ListStoreOrdersRequest{
			StoreId:  uuid.NewString(),
			Page:     -1,
			PageSize: 20,
		},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetStoreOrderUsesStoreScopeAndMapsOrder(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	cartID := uuid.New()
	customerID := uuid.New()

	mock := &getOrderMock{
		result: &use_cases.OrderDetails{
			Order: &entities.Order{
				ID:         orderID,
				CartID:     cartID,
				StoreID:    storeID,
				CustomerID: customerID,
				Currency:   "NGN",
			},
		},
	}
	server := ordergrpc.NewServer(nil, mock, nil, nil, nil, nil)

	response, err := server.GetStoreOrder(
		context.Background(),
		&orderv1.GetStoreOrderRequest{
			StoreId: storeID.String(),
			OrderId: orderID.String(),
		},
	)
	if err != nil {
		t.Fatalf("GetStoreOrder() error = %v", err)
	}
	if response.GetOrder() == nil {
		t.Fatal("GetStoreOrder() returned a nil order")
	}
	if response.GetOrder().GetId() != orderID.String() {
		t.Fatalf("order ID = %q, want %q", response.GetOrder().GetId(), orderID)
	}
	if response.GetOrder().GetStoreId() != storeID.String() {
		t.Fatalf("store ID = %q, want %q", response.GetOrder().GetStoreId(), storeID)
	}
	if mock.input.OrderID != orderID ||
		mock.input.StoreID != storeID ||
		mock.input.UserID != storeID {
		t.Fatalf("store scope was not propagated: %+v", mock.input)
	}
}

func TestGetStoreOrderRejectsNilResult(t *testing.T) {
	server := ordergrpc.NewServer(nil, &getOrderMock{}, nil, nil, nil, nil)

	_, err := server.GetStoreOrder(
		context.Background(),
		&orderv1.GetStoreOrderRequest{
			StoreId: uuid.NewString(),
			OrderId: uuid.NewString(),
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestGetStoreOrderRejectsInvalidOrderID(t *testing.T) {
	server := ordergrpc.NewServer(nil, &getOrderMock{}, nil, nil, nil, nil)

	_, err := server.GetStoreOrder(
		context.Background(),
		&orderv1.GetStoreOrderRequest{
			StoreId: uuid.NewString(),
			OrderId: "invalid",
		},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetStoreOrderMapsUnauthorizedActor(t *testing.T) {
	server := ordergrpc.NewServer(nil, &getOrderMock{err: domain.ErrUnauthorizedActor}, nil, nil, nil, nil)

	_, err := server.GetStoreOrder(
		context.Background(),
		&orderv1.GetStoreOrderRequest{
			StoreId: uuid.NewString(),
			OrderId: uuid.NewString(),
		},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func TestAddShippingPassesStoreScopeAndAmount(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	const shippingAmount int64 = 2500

	mock := &addShippingMock{}
	server := ordergrpc.NewServer(nil, nil, mock, nil, nil, nil)

	response, err := server.AddShipping(
		context.Background(),
		&orderv1.AddShippingRequest{
			StoreId:        storeID.String(),
			OrderId:        orderID.String(),
			ShippingAmount: shippingAmount,
		},
	)
	if err != nil {
		t.Fatalf("AddShipping() error = %v", err)
	}
	if !response.GetSuccess() {
		t.Fatal("AddShipping() success = false, want true")
	}
	if mock.input.StoreID != storeID ||
		mock.input.OrderID != orderID ||
		mock.input.ShippingAmount != shippingAmount {
		t.Fatalf("unexpected AddShipping input: %+v", mock.input)
	}
}

func TestAddShippingRejectsNegativeAmount(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, &addShippingMock{}, nil, nil, nil)

	_, err := server.AddShipping(
		context.Background(),
		&orderv1.AddShippingRequest{
			StoreId:        uuid.NewString(),
			OrderId:        uuid.NewString(),
			ShippingAmount: -1,
		},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestAddShippingMapsInvalidTransition(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, &addShippingMock{err: domain.ErrInvalidTransition}, nil, nil, nil)

	_, err := server.AddShipping(
		context.Background(),
		&orderv1.AddShippingRequest{
			StoreId:        uuid.NewString(),
			OrderId:        uuid.NewString(),
			ShippingAmount: 1000,
		},
	)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestMarkSentPassesStoreAndOrderIDs(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	mock := &markSentMock{}
	server := ordergrpc.NewServer(nil, nil, nil, mock, nil, nil)

	response, err := server.MarkSent(
		context.Background(),
		&orderv1.MarkSentRequest{
			StoreId: storeID.String(),
			OrderId: orderID.String(),
		},
	)
	if err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if !response.GetSuccess() {
		t.Fatal("MarkSent() success = false, want true")
	}
	if mock.input.StoreID != storeID || mock.input.OrderID != orderID {
		t.Fatalf("unexpected MarkSent input: %+v", mock.input)
	}
}

func TestMarkSentMapsInvalidTransition(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, &markSentMock{err: domain.ErrInvalidTransition}, nil, nil)

	_, err := server.MarkSent(
		context.Background(),
		&orderv1.MarkSentRequest{
			StoreId: uuid.NewString(),
			OrderId: uuid.NewString(),
		},
	)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestCancelStoreOrderUsesStoreAsActor(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	mock := &cancelOrderMock{}
	server := ordergrpc.NewServer(nil, nil, nil, nil, mock, nil)

	response, err := server.CancelStoreOrder(
		context.Background(),
		&orderv1.CancelStoreOrderRequest{
			StoreId: storeID.String(),
			OrderId: orderID.String(),
		},
	)
	if err != nil {
		t.Fatalf("CancelStoreOrder() error = %v", err)
	}
	if !response.GetSuccess() {
		t.Fatal("CancelStoreOrder() success = false, want true")
	}
	if mock.input.ActorID != storeID || mock.input.OrderID != orderID {
		t.Fatalf("unexpected cancellation input: %+v", mock.input)
	}
}

func TestCancelStoreOrderMapsUnauthorizedActor(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, nil, &cancelOrderMock{err: domain.ErrUnauthorizedActor}, nil)

	_, err := server.CancelStoreOrder(
		context.Background(),
		&orderv1.CancelStoreOrderRequest{
			StoreId: uuid.NewString(),
			OrderId: uuid.NewString(),
		},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

type initiatePayoutMock struct {
	input  use_cases.InitiatePayoutInput
	output *use_cases.InitiatePayoutOutput
	err    error
}

func (m *initiatePayoutMock) Execute(
	_ context.Context,
	input use_cases.InitiatePayoutInput,
) (*use_cases.InitiatePayoutOutput, error) {
	m.input = input
	return m.output, m.err
}

func TestListOrdersUsesStoreScope(t *testing.T) {
	storeID := uuid.New()
	mock := &listOrdersMock{
		page: &use_cases.OrderPage{Page: 2, PageSize: 10, TotalItems: 15, TotalPages: 2},
	}
	server := ordergrpc.NewServer(mock, nil, nil, nil, nil, nil)

	response, err := server.ListOrders(context.Background(), &orderv1.ListOrdersRequest{
		StoreId:  storeID.String(),
		Page:     2,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListOrders() error = %v", err)
	}
	if response.GetPage() != 2 || response.GetPageSize() != 10 {
		t.Fatalf("unexpected pagination: page=%d page_size=%d", response.GetPage(), response.GetPageSize())
	}
	if response.GetTotalItems() != 15 || response.GetTotalPages() != 2 {
		t.Fatalf("unexpected totals: items=%d pages=%d", response.GetTotalItems(), response.GetTotalPages())
	}
	if mock.input.StoreID != storeID || mock.input.UserID != storeID {
		t.Fatalf("store scope was not propagated: %+v", mock.input)
	}
}

func TestListOrdersRejectsNilRequest(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, nil, nil, nil)

	_, err := server.ListOrders(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestListOrdersRejectsNegativePagination(t *testing.T) {
	server := ordergrpc.NewServer(&listOrdersMock{}, nil, nil, nil, nil, nil)

	_, err := server.ListOrders(context.Background(), &orderv1.ListOrdersRequest{
		StoreId: uuid.NewString(),
		Page:    -1,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetOrderUsesStoreScope(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	mock := &getOrderMock{
		result: &use_cases.OrderDetails{
			Order: &entities.Order{ID: orderID, StoreID: storeID},
		},
	}
	server := ordergrpc.NewServer(nil, mock, nil, nil, nil, nil)

	response, err := server.GetOrder(context.Background(), &orderv1.GetOrderRequest{
		StoreId: storeID.String(),
		OrderId: orderID.String(),
	})
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if response.GetOrder().GetId() != orderID.String() {
		t.Fatalf("order ID = %q, want %q", response.GetOrder().GetId(), orderID)
	}
	if mock.input.OrderID != orderID ||
		mock.input.StoreID != storeID ||
		mock.input.UserID != storeID {
		t.Fatalf("store scope was not propagated: %+v", mock.input)
	}
}

func TestGetOrderRejectsNilRequest(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, nil, nil, nil)

	_, err := server.GetOrder(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestInitiatePayoutPassesStoreScopeAndBeneficiary(t *testing.T) {
	storeID := uuid.New()
	orderID := uuid.New()
	mock := &initiatePayoutMock{
		output: &use_cases.InitiatePayoutOutput{
			PayoutID:      "payout-id",
			AttemptID:     "attempt-id",
			Reference:     "reference",
			Status:        "pending",
			AttemptStatus: "processing",
			Provider:      "paystack",
			AmountMinor:   12500,
			Currency:      "NGN",
		},
	}
	server := ordergrpc.NewServer(nil, nil, nil, nil, nil, mock)

	response, err := server.InitiatePayout(context.Background(), &orderv1.InitiatePayoutRequest{
		OrderId:       orderID.String(),
		StoreId:       storeID.String(),
		AccountName:   "Store Account",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "GTBank",
		CountryCode:   "NG",
	})
	if err != nil {
		t.Fatalf("InitiatePayout() error = %v", err)
	}
	if response.GetPayoutId() != mock.output.PayoutID ||
		response.GetAttemptId() != mock.output.AttemptID ||
		response.GetReference() != mock.output.Reference ||
		response.GetAmountMinor() != mock.output.AmountMinor ||
		response.GetCurrency() != mock.output.Currency {
		t.Fatalf("unexpected payout response: %+v", response)
	}
	if mock.input.OrderID != orderID || mock.input.StoreID != storeID {
		t.Fatalf("unexpected payout scope: %+v", mock.input)
	}
	beneficiary := mock.input.Beneficiary
	if beneficiary.AccountName != "Store Account" ||
		beneficiary.AccountNumber != "0123456789" ||
		beneficiary.BankCode != "058" ||
		beneficiary.BankName != "GTBank" ||
		beneficiary.CountryCode != "NG" {
		t.Fatalf("unexpected beneficiary: %+v", beneficiary)
	}
}

func TestInitiatePayoutRejectsInvalidIDs(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, nil, nil, &initiatePayoutMock{})

	_, err := server.InitiatePayout(context.Background(), &orderv1.InitiatePayoutRequest{
		OrderId: "invalid",
		StoreId: uuid.NewString(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestInitiatePayoutMapsUnauthorizedActor(t *testing.T) {
	server := ordergrpc.NewServer(nil, nil, nil, nil, nil, &initiatePayoutMock{
		err: domain.ErrUnauthorizedActor,
	})

	_, err := server.InitiatePayout(context.Background(), &orderv1.InitiatePayoutRequest{
		OrderId: uuid.NewString(),
		StoreId: uuid.NewString(),
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}
