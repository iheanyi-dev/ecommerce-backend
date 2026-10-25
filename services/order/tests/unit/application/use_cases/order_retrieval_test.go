package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// retrievalOrderRepositoryMock records query scope and returns configured orders.
type retrievalOrderRepositoryMock struct {
	order          *entities.Order
	findErr        error
	customerOrders []*entities.Order
	storeOrders    []*entities.Order
	total          int64
	listErr        error
	customerID     uuid.UUID
	storeID        uuid.UUID
	offset         int
	limit          int
	customerCalls  int
	storeCalls     int
}

func (m *retrievalOrderRepositoryMock) CreateMany(
	_ context.Context, _ []*entities.Order,
) error {
	return nil
}

func (m *retrievalOrderRepositoryMock) FindByID(
	_ context.Context, _ uuid.UUID,
) (*entities.Order, error) {
	return m.order, m.findErr
}

func (m *retrievalOrderRepositoryMock) Update(
	_ context.Context, _ *entities.Order,
) error {
	return nil
}

// ListByCustomerID records customer-scoped pagination arguments.
func (m *retrievalOrderRepositoryMock) ListByCustomerID(
	_ context.Context, id uuid.UUID, offset, limit int,
) ([]*entities.Order, int64, error) {
	m.customerCalls++
	m.customerID, m.offset, m.limit = id, offset, limit
	return m.customerOrders, m.total, m.listErr
}

// ListByStoreID records store-scoped pagination arguments.
func (m *retrievalOrderRepositoryMock) ListByStoreID(
	_ context.Context, id uuid.UUID, offset, limit int,
) ([]*entities.Order, int64, error) {
	m.storeCalls++
	m.storeID, m.offset, m.limit = id, offset, limit
	return m.storeOrders, m.total, m.listErr
}

// retrievalProductRepositoryMock supports batched order-line retrieval.
type retrievalProductRepositoryMock struct {
	lines []*entities.OrderProduct
	err   error
	ids   []uuid.UUID
	calls int
}

func (m *retrievalProductRepositoryMock) CreateMany(
	_ context.Context, _ []*entities.OrderProduct,
) error {
	return nil
}

// FindByOrderIDs records the batch requested by a retrieval use case.
func (m *retrievalProductRepositoryMock) FindByOrderIDs(
	_ context.Context, ids []uuid.UUID,
) ([]*entities.OrderProduct, error) {
	m.calls++
	m.ids = append([]uuid.UUID(nil), ids...)
	return m.lines, m.err
}

// retrievalLoggerMock prevents logging from affecting use-case tests.
type retrievalLoggerMock struct{}

func (retrievalLoggerMock) Log(context.Context, ports.LogEvent) error { return nil }

// retrievalMetricsMock prevents metrics from affecting use-case tests.
type retrievalMetricsMock struct{}

func (retrievalMetricsMock) Increment(context.Context, ports.Metric) error { return nil }
func (retrievalMetricsMock) Observe(context.Context, ports.Metric) error   { return nil }

// newRetrievalOrder creates a valid order for retrieval tests.
func newRetrievalOrder(t *testing.T, storeID, customerID, cartID uuid.UUID) *entities.Order {
	t.Helper()

	order, err := entities.NewOrder(
		storeID, customerID, cartID, "12 Market Road, Lagos", "NGN", 2500,
	)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	return order
}

func TestGetOrderEnforcesCustomerAndStoreScope(t *testing.T) {
	storeID, customerID := uuid.New(), uuid.New()
	order := newRetrievalOrder(t, storeID, customerID, uuid.New())

	tests := []struct {
		name    string
		input   use_cases.GetOrderInput
		wantErr bool
	}{
		{
			name: "customer owns order",
			input: use_cases.GetOrderInput{
				OrderID: order.ID, UserID: customerID,
			},
		},
		{
			name: "matching store scope",
			input: use_cases.GetOrderInput{
				OrderID: order.ID, UserID: uuid.New(), StoreID: storeID,
			},
		},
		{
			name: "wrong customer",
			input: use_cases.GetOrderInput{
				OrderID: order.ID, UserID: uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "wrong store",
			input: use_cases.GetOrderInput{
				OrderID: order.ID, UserID: customerID, StoreID: uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "missing authenticated identity",
			input: use_cases.GetOrderInput{
				OrderID: order.ID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &retrievalOrderRepositoryMock{order: order}
			productRepo := &retrievalProductRepositoryMock{}
			uc := use_cases.NewGetOrder(
				orderRepo, productRepo, retrievalLoggerMock{}, retrievalMetricsMock{},
			)

			got, err := uc.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && got != nil {
				t.Fatal("failed retrieval returned a result")
			}
			if tt.wantErr && productRepo.calls != 0 {
				t.Fatal("product lines were queried before scope validation")
			}
			if !tt.wantErr && (got == nil || got.Order != order) {
				t.Fatal("retrieval did not return the expected order")
			}
		})
	}
}

func TestGetOrderReturnsNoPartialResultOnDependencyFailure(t *testing.T) {
	order := newRetrievalOrder(t, uuid.New(), uuid.New(), uuid.New())
	tests := []struct {
		name        string
		orderRepo   *retrievalOrderRepositoryMock
		productRepo *retrievalProductRepositoryMock
	}{
		{
			name:        "order repository failure",
			orderRepo:   &retrievalOrderRepositoryMock{findErr: errors.New("database unavailable")},
			productRepo: &retrievalProductRepositoryMock{},
		},
		{
			name:        "missing order",
			orderRepo:   &retrievalOrderRepositoryMock{},
			productRepo: &retrievalProductRepositoryMock{},
		},
		{
			name:      "product repository failure",
			orderRepo: &retrievalOrderRepositoryMock{order: order},
			productRepo: &retrievalProductRepositoryMock{
				err: errors.New("database unavailable"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := use_cases.NewGetOrder(
				tt.orderRepo, tt.productRepo, retrievalLoggerMock{}, retrievalMetricsMock{},
			)
			got, err := uc.Execute(context.Background(), use_cases.GetOrderInput{
				OrderID: order.ID, UserID: order.CustomerID,
			})
			if err == nil || got != nil {
				t.Fatalf("Execute() = (%v, %v), want error and no result", got, err)
			}
		})
	}
}

func TestListOrdersUsesStoreScopePaginationAndBatchedProducts(t *testing.T) {
	storeID, customerID := uuid.New(), uuid.New()
	first := newRetrievalOrder(t, storeID, customerID, uuid.New())
	second := newRetrievalOrder(t, storeID, customerID, uuid.New())

	orderRepo := &retrievalOrderRepositoryMock{
		storeOrders: []*entities.Order{first, second},
		total:       9,
	}
	productRepo := &retrievalProductRepositoryMock{
		lines: []*entities.OrderProduct{
			{ID: uuid.New(), OrderID: first.ID, ProductID: uuid.New(), Name: "Keyboard", Qty: 1, Price: 2500},
			{ID: uuid.New(), OrderID: second.ID, ProductID: uuid.New(), Name: "Mouse", Qty: 2, Price: 1000},
		},
	}
	uc := use_cases.NewListOrders(
		orderRepo, productRepo, retrievalLoggerMock{}, retrievalMetricsMock{},
	)

	got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
		UserID: customerID, StoreID: storeID, Page: 2, PageSize: 3,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if orderRepo.storeCalls != 1 || orderRepo.customerCalls != 0 {
		t.Fatalf("listing calls: store=%d customer=%d", orderRepo.storeCalls, orderRepo.customerCalls)
	}
	if orderRepo.storeID != storeID || orderRepo.offset != 3 || orderRepo.limit != 3 {
		t.Fatalf("listing arguments: store=%s offset=%d limit=%d",
			orderRepo.storeID, orderRepo.offset, orderRepo.limit)
	}
	if got.TotalItems != 9 || got.TotalPages != 3 || got.Page != 2 || got.PageSize != 3 {
		t.Fatalf("unexpected pagination metadata: %+v", got)
	}
	if len(got.Items) != 2 ||
		len(got.Items[0].Products) != 1 ||
		len(got.Items[1].Products) != 1 {
		t.Fatalf("product lines were not grouped by order: %+v", got.Items)
	}
	if productRepo.calls != 1 || len(productRepo.ids) != 2 {
		t.Fatalf("expected one batched product query, calls=%d ids=%v",
			productRepo.calls, productRepo.ids)
	}
}

func TestListOrdersUsesCustomerScopeAndDefaultPageSize(t *testing.T) {
	customerID := uuid.New()
	order := newRetrievalOrder(t, uuid.New(), customerID, uuid.New())
	orderRepo := &retrievalOrderRepositoryMock{
		customerOrders: []*entities.Order{order},
		total:          1,
	}
	uc := use_cases.NewListOrders(
		orderRepo, &retrievalProductRepositoryMock{},
		retrievalLoggerMock{}, retrievalMetricsMock{},
	)

	got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
		UserID: customerID, Page: 1,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if orderRepo.customerCalls != 1 || orderRepo.storeCalls != 0 {
		t.Fatalf("listing calls: customer=%d store=%d",
			orderRepo.customerCalls, orderRepo.storeCalls)
	}
	if orderRepo.customerID != customerID ||
		orderRepo.limit != use_cases.DefaultOrderPageSize {
		t.Fatalf("customer listing args: customer=%s limit=%d",
			orderRepo.customerID, orderRepo.limit)
	}
	if got.PageSize != use_cases.DefaultOrderPageSize || got.TotalPages != 1 {
		t.Fatalf("unexpected pagination metadata: %+v", got)
	}
}

func TestListOrdersRejectsInvalidInputBeforeRepositoryCall(t *testing.T) {
	tests := []struct {
		name  string
		input use_cases.ListOrdersInput
	}{
		{name: "missing identity", input: use_cases.ListOrdersInput{Page: 1}},
		{name: "page below one", input: use_cases.ListOrdersInput{UserID: uuid.New(), Page: 0}},
		{name: "negative page size", input: use_cases.ListOrdersInput{
			UserID: uuid.New(), Page: 1, PageSize: -1,
		}},
		{name: "page size exceeds maximum", input: use_cases.ListOrdersInput{
			UserID: uuid.New(), Page: 1, PageSize: use_cases.MaxOrderPageSize + 1,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &retrievalOrderRepositoryMock{}
			uc := use_cases.NewListOrders(
				orderRepo, &retrievalProductRepositoryMock{},
				retrievalLoggerMock{}, retrievalMetricsMock{},
			)
			got, err := uc.Execute(context.Background(), tt.input)
			if err == nil || got != nil {
				t.Fatalf("Execute() = (%v, %v), want validation error", got, err)
			}
			if orderRepo.customerCalls != 0 || orderRepo.storeCalls != 0 {
				t.Fatal("repository called with invalid input")
			}
		})
	}
}

func TestListOrdersHandlesEmptyPageAndProductFailure(t *testing.T) {
	customerID := uuid.New()

	t.Run("empty page", func(t *testing.T) {
		orderRepo := &retrievalOrderRepositoryMock{customerOrders: []*entities.Order{}}
		productRepo := &retrievalProductRepositoryMock{}
		uc := use_cases.NewListOrders(
			orderRepo, productRepo, retrievalLoggerMock{}, retrievalMetricsMock{},
		)
		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, Page: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got == nil || len(got.Items) != 0 || got.TotalPages != 0 {
			t.Fatalf("unexpected empty page: %+v", got)
		}
		if productRepo.calls != 0 {
			t.Fatal("product repository called for empty page")
		}
	})

	t.Run("product retrieval failure", func(t *testing.T) {
		order := newRetrievalOrder(t, uuid.New(), customerID, uuid.New())
		orderRepo := &retrievalOrderRepositoryMock{
			customerOrders: []*entities.Order{order}, total: 1,
		}
		productRepo := &retrievalProductRepositoryMock{err: errors.New("database unavailable")}
		uc := use_cases.NewListOrders(
			orderRepo, productRepo, retrievalLoggerMock{}, retrievalMetricsMock{},
		)
		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, Page: 1,
		})
		if err == nil || got != nil {
			t.Fatalf("Execute() = (%v, %v), want error and no partial result", got, err)
		}
	})
}

func TestListOrdersRejectsPaginationOverflow(t *testing.T) {
	orderRepo := &retrievalOrderRepositoryMock{}
	uc := use_cases.NewListOrders(
		orderRepo, &retrievalProductRepositoryMock{},
		retrievalLoggerMock{}, retrievalMetricsMock{},
	)

	got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
		UserID:   uuid.New(),
		Page:     int(^uint(0) >> 1),
		PageSize: use_cases.MaxOrderPageSize,
	})
	if err == nil || got != nil {
		t.Fatalf("Execute() = (%v, %v), want pagination overflow error", got, err)
	}
	if orderRepo.customerCalls != 0 || orderRepo.storeCalls != 0 {
		t.Fatal("repository called for an overflowing pagination offset")
	}
}

func TestListOrdersRejectsMalformedRepositoryResults(t *testing.T) {
	customerID := uuid.New()
	storeID := uuid.New()

	t.Run("order outside customer scope", func(t *testing.T) {
		order := newRetrievalOrder(t, storeID, uuid.New(), uuid.New())
		repo := &retrievalOrderRepositoryMock{
			customerOrders: []*entities.Order{order}, total: 1,
		}
		products := &retrievalProductRepositoryMock{}
		uc := use_cases.NewListOrders(repo, products, retrievalLoggerMock{}, retrievalMetricsMock{})

		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, Page: 1, PageSize: 1,
		})
		if err == nil || got != nil {
			t.Fatalf("Execute() = (%v, %v), want scope error", got, err)
		}
		if products.calls != 0 {
			t.Fatal("product repository called for an out-of-scope order")
		}
	})

	t.Run("order outside store scope", func(t *testing.T) {
		order := newRetrievalOrder(t, uuid.New(), customerID, uuid.New())
		repo := &retrievalOrderRepositoryMock{
			storeOrders: []*entities.Order{order}, total: 1,
		}
		uc := use_cases.NewListOrders(repo, &retrievalProductRepositoryMock{},
			retrievalLoggerMock{}, retrievalMetricsMock{})

		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, StoreID: storeID, Page: 1, PageSize: 1,
		})
		if err == nil || got != nil {
			t.Fatalf("Execute() = (%v, %v), want scope error", got, err)
		}
	})

	t.Run("negative total", func(t *testing.T) {
		repo := &retrievalOrderRepositoryMock{total: -1}
		uc := use_cases.NewListOrders(repo, &retrievalProductRepositoryMock{},
			retrievalLoggerMock{}, retrievalMetricsMock{})

		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, Page: 1, PageSize: 1,
		})
		if err == nil || got != nil {
			t.Fatalf("Execute() = (%v, %v), want invalid-total error", got, err)
		}
	})

	t.Run("too many orders for page size", func(t *testing.T) {
		first := newRetrievalOrder(t, storeID, customerID, uuid.New())
		second := newRetrievalOrder(t, storeID, customerID, uuid.New())
		repo := &retrievalOrderRepositoryMock{
			customerOrders: []*entities.Order{first, second}, total: 2,
		}
		uc := use_cases.NewListOrders(repo, &retrievalProductRepositoryMock{},
			retrievalLoggerMock{}, retrievalMetricsMock{})

		got, err := uc.Execute(context.Background(), use_cases.ListOrdersInput{
			UserID: customerID, Page: 1, PageSize: 1,
		})
		if err == nil || got != nil {
			t.Fatalf("Execute() = (%v, %v), want oversized-page error", got, err)
		}
	})
}

func TestRetrievalRejectsMalformedProductLines(t *testing.T) {
	customerID := uuid.New()
	order := newRetrievalOrder(t, uuid.New(), customerID, uuid.New())

	tests := []struct {
		name string
		line *entities.OrderProduct
	}{
		{
			name: "nil product line",
		},
		{
			name: "unexpected order ID",
			line: &entities.OrderProduct{
				ID: uuid.New(), OrderID: uuid.New(), ProductID: uuid.New(),
				Name: "Keyboard", Qty: 1, Price: 2500,
			},
		},
		{
			name: "missing product ID",
			line: &entities.OrderProduct{
				ID: uuid.New(), OrderID: order.ID,
				Name: "Keyboard", Qty: 1, Price: 2500,
			},
		},
		{
			name: "invalid quantity",
			line: &entities.OrderProduct{
				ID: uuid.New(), OrderID: order.ID, ProductID: uuid.New(),
				Name: "Keyboard", Qty: 0, Price: 2500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &retrievalOrderRepositoryMock{order: order}
			productRepo := &retrievalProductRepositoryMock{lines: []*entities.OrderProduct{tt.line}}
			uc := use_cases.NewGetOrder(orderRepo, productRepo,
				retrievalLoggerMock{}, retrievalMetricsMock{})

			got, err := uc.Execute(context.Background(), use_cases.GetOrderInput{
				OrderID: order.ID, UserID: customerID,
			})
			if err == nil || got != nil {
				t.Fatalf("Execute() = (%v, %v), want malformed-product error", got, err)
			}
		})
	}
}
