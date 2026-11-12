package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

type productRepositoryFake struct {
	product        *entities.Product
	getErr         error
	createErr      error
	createdProduct *entities.Product
	createCalls    int
	updateErr      error
	updateCalls    int
	updatedProduct *entities.Product
	deleteErr      error
	deleteCalls    int
	deletedID      string
	gotID          string
	getCalls       int
	productsByIDs  []*entities.Product
	getByIDsErr    error
	getByIDsCalls  int
	gotIDs         []string

	products           []*entities.Product
	total              int64
	listByStoreErr     error
	listByStoreCalls   int
	listLimit          int
	listOffset         int
	countByStore       int64
	countByStoreErr    error
	countByStoreCalls  int
	listProductsErr    error
	listProductsCalls  int
	listFilters        ports.ProductListFilters
	listProductsLimit  int
	listProductsOffset int
}

func (f *productRepositoryFake) Create(_ context.Context, product *entities.Product) error {
	f.createCalls++
	f.createdProduct = product
	return f.createErr
}

func (f *productRepositoryFake) GetByID(_ context.Context, productID string) (*entities.Product, error) {
	f.getCalls++
	f.gotID = productID
	return f.product, f.getErr
}

func (f *productRepositoryFake) GetByIDs(
	_ context.Context,
	productIDs []string,
) ([]*entities.Product, error) {
	f.getByIDsCalls++
	f.gotIDs = append([]string(nil), productIDs...)
	return f.productsByIDs, f.getByIDsErr
}

func (f *productRepositoryFake) Update(_ context.Context, product *entities.Product) error {
	f.updateCalls++
	f.updatedProduct = product
	return f.updateErr
}

func (f *productRepositoryFake) Delete(_ context.Context, productID string) error {
	f.deleteCalls++
	f.deletedID = productID
	return f.deleteErr
}

func (f *productRepositoryFake) ListByStore(
	_ context.Context,
	_ string,
	limit, offset int,
) ([]*entities.Product, int64, error) {
	f.listByStoreCalls++
	f.listLimit = limit
	f.listOffset = offset
	return f.products, f.total, f.listByStoreErr
}

func (f *productRepositoryFake) CountByStore(
	_ context.Context,
	_ string,
) (int64, error) {
	f.countByStoreCalls++
	return f.countByStore, f.countByStoreErr
}

func (f *productRepositoryFake) ListProducts(
	_ context.Context,
	filters ports.ProductListFilters,
	limit, offset int,
) ([]*entities.Product, int64, error) {
	f.listProductsCalls++
	f.listFilters = filters
	f.listProductsLimit = limit
	f.listProductsOffset = offset
	return f.products, f.total, f.listProductsErr
}

// SearchByStore satisfies the unified ProductRepository contract.
func (f *productRepositoryFake) SearchByStore(
	_ context.Context,
	_ string,
	_ string,
	_ ports.StoreProductSearchFilters,
) ([]*entities.Product, error) {
	return f.products, nil
}

type storeClientFake struct {
	ownerStoreID     string
	ownerLookupErr   error
	ownerLookupCalls int
	gotOwnerID       string
	active           bool
	checkErr         error
	incrementErr     error
	decrementErr     error
	gotStoreID       string
	checkCalls       int
	incrementCalls   int
	decrementCalls   int
	incrementStore   string
	decrementStore   string
	activeStoreIDs   []string
	filteredStoreIDs []string
	filterActiveErr  error
}

func (f *storeClientFake) GetStoreIDByOwnerID(
	_ context.Context,
	ownerID string,
) (string, error) {
	f.ownerLookupCalls++
	f.gotOwnerID = ownerID
	return f.ownerStoreID, f.ownerLookupErr
}

func (f *storeClientFake) IncrementProductCount(_ context.Context, storeID string) error {
	f.incrementCalls++
	f.incrementStore = storeID
	return f.incrementErr
}

func (f *storeClientFake) DecrementProductCount(_ context.Context, storeID string) error {
	f.decrementCalls++
	f.decrementStore = storeID
	return f.decrementErr
}

func (f *storeClientFake) IsStoreActive(_ context.Context, storeID string) (bool, error) {
	f.checkCalls++
	f.gotStoreID = storeID
	return f.active, f.checkErr
}

func (f *storeClientFake) FilterActiveStoreIDs(
	_ context.Context,
	storeIDs []string,
) ([]string, error) {
	f.filteredStoreIDs = append([]string(nil), storeIDs...)
	if f.filterActiveErr != nil {
		return nil, f.filterActiveErr
	}
	if f.activeStoreIDs != nil {
		return append([]string(nil), f.activeStoreIDs...), nil
	}
	return append([]string(nil), storeIDs...), nil
}

type loggerFake struct {
	events []ports.LogEvent
	err    error
}

func (f *loggerFake) Log(_ context.Context, event ports.LogEvent) error {
	f.events = append(f.events, event)
	return f.err
}

type metricsFake struct {
	metrics []ports.Metric
	err     error
}

func (f *metricsFake) Increment(_ context.Context, metric ports.Metric) error {
	f.metrics = append(f.metrics, metric)
	return f.err
}

func (f *metricsFake) Observe(context.Context, ports.Metric) error {
	return nil
}

func newTestProduct(t *testing.T) *entities.Product {
	t.Helper()

	price, err := valueobjects.NewMoney(150000, "NGN")
	if err != nil {
		t.Fatalf("create test price: %v", err)
	}

	product, err := entities.NewProduct(
		"store-123",
		"Test product",
		"Product description",
		price,
		3,
	)
	if err != nil {
		t.Fatalf("create test product: %v", err)
	}

	return product
}

func TestGetProductByID(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	storeErr := errors.New("store service unavailable")

	tests := []struct {
		name              string
		repositoryProduct *entities.Product
		repositoryErr     error
		storeActive       bool
		storeErr          error
		wantErr           error
		wantProduct       bool
		wantStoreCheck    bool
		wantEvent         string
		wantFailure       string
	}{
		{
			name:              "returns product when store is active",
			repositoryProduct: newTestProduct(t),
			storeActive:       true,
			wantProduct:       true,
			wantStoreCheck:    true,
			wantEvent:         "product.get_by_id.succeeded",
		},
		{
			name:           "returns not found when product is missing",
			wantErr:        appusecases.ErrProductNotFound,
			wantStoreCheck: false,
			wantEvent:      "product.get_by_id.not_found",
		},
		{
			name:              "hides product when store is inactive",
			repositoryProduct: newTestProduct(t),
			storeActive:       false,
			wantErr:           appusecases.ErrProductNotFound,
			wantStoreCheck:    true,
			wantEvent:         "product.get_by_id.store_inactive",
		},
		{
			name:           "propagates repository failure",
			repositoryErr:  repositoryErr,
			wantErr:        repositoryErr,
			wantStoreCheck: false,
			wantEvent:      "product.get_by_id.repository_failed",
			wantFailure:    "persistence",
		},
		{
			name:              "propagates store activity check failure",
			repositoryProduct: newTestProduct(t),
			storeErr:          storeErr,
			wantErr:           storeErr,
			wantStoreCheck:    true,
			wantEvent:         "product.get_by_id.store_check_failed",
			wantFailure:       "store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				product: tt.repositoryProduct,
				getErr:  tt.repositoryErr,
			}
			store := &storeClientFake{
				active:   tt.storeActive,
				checkErr: tt.storeErr,
			}
			logger := &loggerFake{}
			metrics := &metricsFake{}

			useCase := appusecases.NewGetProductByID(repository, store, logger, metrics, nil)
			got, err := useCase.Execute(context.Background(), "product-456")

			if tt.wantErr == nil && err != nil {
				t.Fatalf("Execute() unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want error wrapping %v", err, tt.wantErr)
			}
			if (got.Product != nil) != tt.wantProduct {
				t.Fatalf("Execute() returned product = %v, want product present = %v", got.Product != nil, tt.wantProduct)
			}
			if repository.gotID != "product-456" {
				t.Errorf("repository received product ID %q", repository.gotID)
			}
			if (store.checkCalls > 0) != tt.wantStoreCheck {
				t.Errorf("store check calls = %d, want check = %v", store.checkCalls, tt.wantStoreCheck)
			}
			if tt.wantStoreCheck && tt.repositoryProduct != nil && store.gotStoreID != tt.repositoryProduct.StoreID() {
				t.Errorf("store received store ID %q, want %q", store.gotStoreID, tt.repositoryProduct.StoreID())
			}

			if len(logger.events) != 1 {
				t.Fatalf("logged events = %d, want 1", len(logger.events))
			}
			if logger.events[0].Event != tt.wantEvent {
				t.Errorf("logged event = %q, want %q", logger.events[0].Event, tt.wantEvent)
			}
			if logger.events[0].Operation != "get_product_by_id" {
				t.Errorf("logged operation = %q", logger.events[0].Operation)
			}
			if logger.events[0].FailureCategory != tt.wantFailure {
				t.Errorf("failure category = %q, want %q", logger.events[0].FailureCategory, tt.wantFailure)
			}

			if len(metrics.metrics) != 1 {
				t.Fatalf("metrics recorded = %d, want 1", len(metrics.metrics))
			}
			metric := metrics.metrics[0]
			if metric.Name != "product_operations_total" || metric.Value != 1 {
				t.Errorf("unexpected metric: %+v", metric)
			}
			if metric.Labels["operation"] != "get_product_by_id" || metric.Labels["event"] != tt.wantEvent {
				t.Errorf("unexpected metric labels: %+v", metric.Labels)
			}
		})
	}
}

func TestGetProductByIDObservabilityFailuresDoNotChangeResult(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{active: true}
	logger := &loggerFake{err: errors.New("logger unavailable")}
	metrics := &metricsFake{err: errors.New("metrics unavailable")}

	useCase := appusecases.NewGetProductByID(repository, store, logger, metrics, nil)
	got, err := useCase.Execute(context.Background(), product.ID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Product != product {
		t.Fatal("Execute() did not return the retrieved product")
	}
	if len(logger.events) != 1 {
		t.Errorf("logger calls = %d, want 1", len(logger.events))
	}
	if len(metrics.metrics) != 1 {
		t.Errorf("metrics calls = %d, want 1", len(metrics.metrics))
	}
}
