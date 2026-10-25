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

type cartClientMock struct {
	output *ports.CartCheckoutOutput
	err    error
	calls  int
}

func (m *cartClientMock) Checkout(
	_ context.Context,
	_, _ uuid.UUID,
) (*ports.CartCheckoutOutput, error) {
	m.calls++
	return m.output, m.err
}

type loggerMock struct{}

func (loggerMock) Log(context.Context, ports.LogEvent) error { return nil }

type metricsMock struct{}

func (metricsMock) Increment(context.Context, ports.Metric) error { return nil }
func (metricsMock) Observe(context.Context, ports.Metric) error   { return nil }

type transactionMock struct {
	orders            []*entities.Order
	products          []*entities.OrderProduct
	failOrderAt       int
	failProductAt     int
	transactions      int
	orderBatchCalls   int
	productBatchCalls int
	rolledBack        bool
}

func (m *transactionMock) WithinTransaction(
	_ context.Context,
	fn func(ports.OrderRepository, ports.OrderProductRepository) error,
) error {
	m.transactions++

	orderRepo := &orderRepositoryMock{tx: m}
	productRepo := &orderProductRepositoryMock{tx: m}

	err := fn(orderRepo, productRepo)
	if err != nil {
		m.rolledBack = true
		m.orders = nil
		m.products = nil
		return err
	}
	return nil
}

type orderRepositoryMock struct {
	tx *transactionMock
}

// FindByID is unused by creation tests and satisfies the shared repository port.
func (m *orderRepositoryMock) FindByID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Order, error) {
	return nil, errors.New("FindByID is not configured in creation tests")
}

// Update is unused by creation tests and satisfies the shared repository port.
func (m *orderRepositoryMock) Update(
	_ context.Context,
	_ *entities.Order,
) error {
	return errors.New("Update is not configured in creation tests")
}

func (m *orderRepositoryMock) CreateMany(
	_ context.Context,
	orders []*entities.Order,
) error {
	m.tx.orderBatchCalls++
	if m.tx.failOrderAt > 0 && len(orders) >= m.tx.failOrderAt {
		return errors.New("order persistence failed")
	}
	m.tx.orders = append(m.tx.orders, orders...)
	return nil
}

type orderProductRepositoryMock struct {
	tx *transactionMock
}

func (m *orderProductRepositoryMock) CreateMany(
	_ context.Context,
	products []*entities.OrderProduct,
) error {
	m.tx.productBatchCalls++
	if m.tx.failProductAt > 0 && len(products) >= m.tx.failProductAt {
		return errors.New("order product persistence failed")
	}
	m.tx.products = append(m.tx.products, products...)
	return nil
}

func TestCreateOrderCreatesOneOrderPerStore(t *testing.T) {
	customerID := uuid.New()
	cartID := uuid.New()
	storeA := uuid.New()
	storeB := uuid.New()
	productA := uuid.New()
	productB := uuid.New()
	productC := uuid.New()

	cart := &cartClientMock{
		output: &ports.CartCheckoutOutput{
			CartID: cartID,
			Stores: []ports.CartCheckoutStore{
				{
					StoreID: storeA,
					Items: []ports.CartCheckoutItem{
						{ProductID: productA, Name: "Keyboard", Quantity: 2, PriceMinor: 1500, Currency: "NGN"},
						{ProductID: productB, Name: "Mouse", Quantity: 1, PriceMinor: 500, Currency: "NGN"},
					},
				},
				{
					StoreID: storeB,
					Items: []ports.CartCheckoutItem{
						{ProductID: productC, Name: "Monitor", Quantity: 1, PriceMinor: 25000, Currency: "NGN"},
					},
				},
			},
		},
	}
	tx := &transactionMock{}

	uc := use_cases.NewCreateOrder(cart, tx, loggerMock{}, metricsMock{})
	orders, err := uc.Execute(context.Background(), use_cases.CreateOrderInput{
		CartID:           cartID,
		CustomerID:       customerID,
		DeliveryLocation: "Lagos",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	if len(tx.orders) != 2 {
		t.Fatalf("expected 2 persisted orders, got %d", len(tx.orders))
	}
	if len(tx.products) != 3 {
		t.Fatalf("expected 3 persisted order products, got %d", len(tx.products))
	}
	if tx.transactions != 1 {
		t.Fatalf("expected one transaction, got %d", tx.transactions)
	}
	if tx.orderBatchCalls != 1 {
		t.Fatalf("expected one order batch call, got %d", tx.orderBatchCalls)
	}
	if tx.productBatchCalls != 1 {
		t.Fatalf("expected one order-product batch call, got %d", tx.productBatchCalls)
	}

	totals := map[uuid.UUID]int64{}
	for _, order := range orders {
		totals[order.StoreID] = order.PriceProduct
		if order.CustomerID != customerID {
			t.Fatalf("order customer ID = %s, want %s", order.CustomerID, customerID)
		}
		if order.CartID != cartID {
			t.Fatalf("order cart ID = %s, want %s", order.CartID, cartID)
		}
	}
	if totals[storeA] != 3500 {
		t.Fatalf("store A subtotal = %d, want 3500", totals[storeA])
	}
	if totals[storeB] != 25000 {
		t.Fatalf("store B subtotal = %d, want 25000", totals[storeB])
	}
	if cart.calls != 1 {
		t.Fatalf("expected one cart checkout, got %d", cart.calls)
	}
}

func TestCreateOrderRejectsInvalidCheckoutWithoutPersistence(t *testing.T) {
	customerID := uuid.New()
	cartID := uuid.New()

	tests := []struct {
		name   string
		output *ports.CartCheckoutOutput
	}{
		{
			name:   "nil checkout",
			output: nil,
		},
		{
			name: "empty checkout",
			output: &ports.CartCheckoutOutput{
				CartID: cartID,
			},
		},
		{
			name: "cart ID mismatch",
			output: &ports.CartCheckoutOutput{
				CartID: uuid.New(),
				Stores: []ports.CartCheckoutStore{{
					StoreID: uuid.New(),
					Items: []ports.CartCheckoutItem{{
						ProductID: uuid.New(), Name: "Item", Quantity: 1,
						PriceMinor: 100, Currency: "NGN",
					}},
				}},
			},
		},
		{
			name: "invalid quantity",
			output: &ports.CartCheckoutOutput{
				CartID: cartID,
				Stores: []ports.CartCheckoutStore{{
					StoreID: uuid.New(),
					Items: []ports.CartCheckoutItem{{
						ProductID: uuid.New(), Name: "Item", Quantity: 0,
						PriceMinor: 100, Currency: "NGN",
					}},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localTx := &transactionMock{}
			uc := use_cases.NewCreateOrder(
				&cartClientMock{output: tt.output},
				localTx,
				loggerMock{},
				metricsMock{},
			)

			_, err := uc.Execute(context.Background(), use_cases.CreateOrderInput{
				CartID: cartID, CustomerID: customerID, DeliveryLocation: "Lagos",
			})
			if err == nil {
				t.Fatal("expected validation error")
			}
			if localTx.transactions != 0 {
				t.Fatal("persistence transaction started for invalid checkout")
			}
		})
	}
}

func TestCreateOrderRollsBackWhenProductPersistenceFails(t *testing.T) {
	customerID := uuid.New()
	cartID := uuid.New()
	tx := &transactionMock{failProductAt: 2}

	cart := &cartClientMock{
		output: &ports.CartCheckoutOutput{
			CartID: cartID,
			Stores: []ports.CartCheckoutStore{
				{
					StoreID: uuid.New(),
					Items: []ports.CartCheckoutItem{
						{ProductID: uuid.New(), Name: "First", Quantity: 1, PriceMinor: 100, Currency: "NGN"},
						{ProductID: uuid.New(), Name: "Second", Quantity: 1, PriceMinor: 200, Currency: "NGN"},
					},
				},
				{
					StoreID: uuid.New(),
					Items: []ports.CartCheckoutItem{
						{ProductID: uuid.New(), Name: "Third", Quantity: 1, PriceMinor: 300, Currency: "NGN"},
					},
				},
			},
		},
	}

	uc := use_cases.NewCreateOrder(cart, tx, loggerMock{}, metricsMock{})
	_, err := uc.Execute(context.Background(), use_cases.CreateOrderInput{
		CartID: cartID, CustomerID: customerID, DeliveryLocation: "Lagos",
	})
	if err == nil {
		t.Fatal("expected persistence error")
	}
	if !tx.rolledBack {
		t.Fatal("expected transaction rollback")
	}
	if len(tx.orders) != 0 || len(tx.products) != 0 {
		t.Fatal("expected no persisted records after rollback")
	}
	if tx.orderBatchCalls != 1 || tx.productBatchCalls != 1 {
		t.Fatalf("expected one call per batch repository, got orders=%d products=%d",
			tx.orderBatchCalls, tx.productBatchCalls)
	}
}

func TestCreateOrderRejectsSubtotalOverflowBeforeTransaction(t *testing.T) {
	cartID := uuid.New()
	tx := &transactionMock{}
	cart := &cartClientMock{
		output: &ports.CartCheckoutOutput{
			CartID: cartID,
			Stores: []ports.CartCheckoutStore{{
				StoreID: uuid.New(),
				Items: []ports.CartCheckoutItem{{
					ProductID:  uuid.New(),
					Name:       "Expensive item",
					Quantity:   2,
					PriceMinor: int64(^uint64(0) >> 1),
					Currency:   "NGN",
				}},
			}},
		},
	}

	uc := use_cases.NewCreateOrder(cart, tx, loggerMock{}, metricsMock{})
	_, err := uc.Execute(context.Background(), use_cases.CreateOrderInput{
		CartID: cartID, CustomerID: uuid.New(), DeliveryLocation: "Lagos",
	})
	if err == nil {
		t.Fatal("expected subtotal overflow validation error")
	}
	if tx.transactions != 0 {
		t.Fatal("transaction started for overflowing subtotal")
	}
}

func TestCreateOrderRejectsMixedCurrenciesBeforeTransaction(t *testing.T) {
	cartID := uuid.New()
	tx := &transactionMock{}
	cart := &cartClientMock{
		output: &ports.CartCheckoutOutput{
			CartID: cartID,
			Stores: []ports.CartCheckoutStore{{
				StoreID: uuid.New(),
				Items: []ports.CartCheckoutItem{
					{
						ProductID: uuid.New(), Name: "First", Quantity: 1,
						PriceMinor: 100, Currency: "NGN",
					},
					{
						ProductID: uuid.New(), Name: "Second", Quantity: 1,
						PriceMinor: 200, Currency: "USD",
					},
				},
			}},
		},
	}

	uc := use_cases.NewCreateOrder(cart, tx, loggerMock{}, metricsMock{})
	_, err := uc.Execute(context.Background(), use_cases.CreateOrderInput{
		CartID: cartID, CustomerID: uuid.New(), DeliveryLocation: "Lagos",
	})
	if err == nil {
		t.Fatal("expected mixed-currency validation error")
	}
	if tx.transactions != 0 {
		t.Fatal("transaction started for mixed currencies")
	}
}

// ListByCustomerID is unused by creation tests and returns an empty page.
func (m *orderRepositoryMock) ListByCustomerID(
	_ context.Context,
	_ uuid.UUID,
	_, _ int,
) ([]*entities.Order, int64, error) {
	return nil, 0, nil
}

// ListByStoreID is unused by creation tests and returns an empty page.
func (m *orderRepositoryMock) ListByStoreID(
	_ context.Context,
	_ uuid.UUID,
	_, _ int,
) ([]*entities.Order, int64, error) {
	return nil, 0, nil
}

// FindByOrderIDs is unused by creation tests and returns no product lines.
func (m *orderProductRepositoryMock) FindByOrderIDs(
	_ context.Context,
	_ []uuid.UUID,
) ([]*entities.OrderProduct, error) {
	return nil, nil
}
