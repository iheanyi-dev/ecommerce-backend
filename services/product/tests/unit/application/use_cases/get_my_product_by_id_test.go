package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestGetMyProductByID(t *testing.T) {
	storeErr := errors.New("store service unavailable")
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name                string
		ownerStoreID        string
		ownerLookupErr      error
		repositoryProduct   *entities.Product
		repositoryErr       error
		wantErr             error
		wantProduct         bool
		wantRepositoryCalls int
		wantEvent           string
		wantFailure         string
	}{
		{
			name:                "returns product owned by vendor",
			ownerStoreID:        "store-123",
			repositoryProduct:   newTestProduct(t),
			wantProduct:         true,
			wantRepositoryCalls: 1,
			wantEvent:           "product.get_my_product_by_id.succeeded",
		},
		{
			name:                "returns not found when owner has no store",
			wantErr:             appusecases.ErrProductNotFound,
			wantRepositoryCalls: 0,
			wantEvent:           "product.get_my_product_by_id.not_found",
		},
		{
			name:                "returns not found when product does not exist",
			ownerStoreID:        "store-123",
			wantErr:             appusecases.ErrProductNotFound,
			wantRepositoryCalls: 1,
			wantEvent:           "product.get_my_product_by_id.not_found",
		},
		{
			name:                "hides product belonging to another store",
			ownerStoreID:        "store-another",
			repositoryProduct:   newTestProduct(t),
			wantErr:             appusecases.ErrProductNotFound,
			wantRepositoryCalls: 1,
			wantEvent:           "product.get_my_product_by_id.ownership_mismatch",
		},
		{
			name:                "propagates store lookup failure",
			ownerLookupErr:      storeErr,
			wantErr:             storeErr,
			wantRepositoryCalls: 0,
			wantEvent:           "product.get_my_product_by_id.store_lookup_failed",
			wantFailure:         "store",
		},
		{
			name:                "propagates repository failure",
			ownerStoreID:        "store-123",
			repositoryErr:       repositoryErr,
			wantErr:             repositoryErr,
			wantRepositoryCalls: 1,
			wantEvent:           "product.get_my_product_by_id.repository_failed",
			wantFailure:         "persistence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				product: tt.repositoryProduct,
				getErr:  tt.repositoryErr,
			}
			store := &storeClientFake{
				ownerStoreID:   tt.ownerStoreID,
				ownerLookupErr: tt.ownerLookupErr,
			}
			logger := &loggerFake{}
			metrics := &metricsFake{}

			useCase := appusecases.NewGetMyProductByID(repository, store, logger, metrics, nil)
			got, err := useCase.Execute(context.Background(), "owner-123", "product-456")

			if tt.wantErr == nil && err != nil {
				t.Fatalf("Execute() unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want error wrapping %v", err, tt.wantErr)
			}
			if (got.Product != nil) != tt.wantProduct {
				t.Errorf("product present = %v, want %v", got.Product != nil, tt.wantProduct)
			}
			if store.ownerLookupCalls != 1 {
				t.Errorf("owner store lookups = %d, want 1", store.ownerLookupCalls)
			}
			if store.gotOwnerID != "owner-123" {
				t.Errorf("owner ID = %q, want owner-123", store.gotOwnerID)
			}
			if repository.getCalls != tt.wantRepositoryCalls {
				t.Errorf("repository calls = %d, want %d", repository.getCalls, tt.wantRepositoryCalls)
			}
			if repository.getCalls > 0 && repository.gotID != "product-456" {
				t.Errorf("repository product ID = %q, want product-456", repository.gotID)
			}

			if len(logger.events) != 1 {
				t.Fatalf("logged events = %d, want 1", len(logger.events))
			}
			event := logger.events[0]
			if event.Event != tt.wantEvent {
				t.Errorf("logged event = %q, want %q", event.Event, tt.wantEvent)
			}
			if event.Operation != "get_my_product_by_id" {
				t.Errorf("logged operation = %q", event.Operation)
			}
			if event.FailureCategory != tt.wantFailure {
				t.Errorf("failure category = %q, want %q", event.FailureCategory, tt.wantFailure)
			}

			if len(metrics.metrics) != 1 {
				t.Fatalf("metrics recorded = %d, want 1", len(metrics.metrics))
			}
			metric := metrics.metrics[0]
			if metric.Name != "product_operations_total" || metric.Value != 1 {
				t.Errorf("unexpected metric: %+v", metric)
			}
			if metric.Labels["operation"] != "get_my_product_by_id" ||
				metric.Labels["event"] != tt.wantEvent {
				t.Errorf("unexpected metric labels: %+v", metric.Labels)
			}
		})
	}
}

func TestGetMyProductByIDObservabilityFailuresDoNotChangeResult(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID()}
	logger := &loggerFake{err: errors.New("logger unavailable")}
	metrics := &metricsFake{err: errors.New("metrics unavailable")}

	useCase := appusecases.NewGetMyProductByID(repository, store, logger, metrics, nil)
	got, err := useCase.Execute(context.Background(), "owner-123", product.ID())
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

	_ = ports.Metric{} // Keep the port contract visible in this test package.
}
