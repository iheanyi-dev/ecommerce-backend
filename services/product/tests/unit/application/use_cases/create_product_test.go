package use_cases_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

type embeddingDispatcherFake struct {
	dispatchErr       error
	dispatchCalls     int
	dispatchedProduct ports.ProductEmbedding
	deletionErr       error
	deletionCalls     int
	deletedProductID  string
	deletedStoreID    string
}

func (f *embeddingDispatcherFake) DispatchProductEmbedding(
	_ context.Context,
	product ports.ProductEmbedding,
) error {
	f.dispatchCalls++
	f.dispatchedProduct = product
	return f.dispatchErr
}

func (f *embeddingDispatcherFake) DispatchProductEmbeddingDeletion(
	_ context.Context,
	productID string,
	storeID string,
) error {
	f.deletionCalls++
	f.deletedProductID = productID
	f.deletedStoreID = storeID
	return f.deletionErr
}

func TestCreateProduct(t *testing.T) {
	persistenceErr := errors.New("database unavailable")
	capacityErr := errors.New("store capacity reached")
	compensationErr := errors.New("store service unavailable")
	embeddingErr := errors.New("embedding broker unavailable")

	tests := []struct {
		name                 string
		input                appusecases.CreateProductInput
		capacityErr          error
		storeLookupErr       error
		persistenceErr       error
		compensationErr      error
		embeddingErr         error
		wantErr              error
		wantErrorText        string
		wantProduct          bool
		wantCreateCalls      int
		wantIncrementCalls   int
		wantDecrementCalls   int
		wantEmbeddingCalls   int
		wantEvent            string
		wantFailureCategory  string
		wantAdditionalEvents []string
	}{
		{
			name: "creates product and dispatches embedding",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			wantProduct:        true,
			wantCreateCalls:    1,
			wantIncrementCalls: 1,
			wantEmbeddingCalls: 1,
			wantEvent:          "product.create.succeeded",
		},
		{
			name: "validation failure does not reserve or persist",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			wantErrorText:       "create product entity",
			wantCreateCalls:     0,
			wantIncrementCalls:  0,
			wantDecrementCalls:  0,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.validation_failed",
			wantFailureCategory: "validation",
		},
		{
			name: "owner without a store cannot create a product",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			wantErr:             appusecases.ErrProductOwnerStoreNotFound,
			wantCreateCalls:     0,
			wantIncrementCalls:  0,
			wantDecrementCalls:  0,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.store_not_found",
			wantFailureCategory: "validation",
		},
		{
			name: "owner store lookup failure does not reserve or persist",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			storeLookupErr:      errors.New("store service unavailable"),
			wantErrorText:       "resolve owner store",
			wantCreateCalls:     0,
			wantIncrementCalls:  0,
			wantDecrementCalls:  0,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.store_lookup_failed",
			wantFailureCategory: "store",
		},
		{
			name: "capacity reservation failure does not persist",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			capacityErr:         capacityErr,
			wantErrorText:       "reserve store product capacity",
			wantCreateCalls:     0,
			wantIncrementCalls:  1,
			wantDecrementCalls:  0,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.capacity_reservation_failed",
			wantFailureCategory: "store",
		},
		{
			name: "persistence failure compensates capacity reservation",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			persistenceErr:      persistenceErr,
			wantErr:             persistenceErr,
			wantCreateCalls:     1,
			wantIncrementCalls:  1,
			wantDecrementCalls:  1,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.persistence_failed",
			wantFailureCategory: "persistence",
		},
		{
			name: "unknown persistence outcome does not compensate",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			persistenceErr:      appusecases.ErrProductCreationOutcomeUnknown,
			wantErr:             appusecases.ErrProductCreationOutcomeUnknown,
			wantErrorText:       "outcome unknown",
			wantCreateCalls:     1,
			wantIncrementCalls:  1,
			wantDecrementCalls:  0,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.outcome_unknown",
			wantFailureCategory: "persistence",
		},
		{
			name: "compensation failure reports both failures",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			persistenceErr:      persistenceErr,
			compensationErr:     compensationErr,
			wantErr:             persistenceErr,
			wantErrorText:       "release reserved store capacity",
			wantCreateCalls:     1,
			wantIncrementCalls:  1,
			wantDecrementCalls:  1,
			wantEmbeddingCalls:  0,
			wantEvent:           "product.create.capacity_compensation_failed",
			wantFailureCategory: "compensation",
		},
		{
			wantAdditionalEvents: []string{"product.create.embedding_dispatch_failed"},
			name:                 "embedding dispatch failure does not fail product creation",
			input: appusecases.CreateProductInput{
				OwnerID:     "owner-123",
				Name:        "Keyboard",
				Description: "Mechanical keyboard",
				Price:       mustMoney(t, 250000, "NGN"),
				Quantity:    5,
			},
			embeddingErr:       embeddingErr,
			wantProduct:        true,
			wantCreateCalls:    1,
			wantIncrementCalls: 1,
			wantDecrementCalls: 0,
			wantEmbeddingCalls: 1,
			wantEvent:          "product.create.succeeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{createErr: tt.persistenceErr}
			ownerStoreID := "store-123"
			if tt.wantErr == appusecases.ErrProductOwnerStoreNotFound {
				ownerStoreID = ""
			}
			store := &storeClientFake{
				ownerStoreID:   ownerStoreID,
				ownerLookupErr: tt.storeLookupErr,
				incrementErr:   tt.capacityErr,
				decrementErr:   tt.compensationErr,
			}
			embeddings := &embeddingDispatcherFake{dispatchErr: tt.embeddingErr}
			logger := &loggerFake{}
			metrics := &metricsFake{}

			useCase := appusecases.NewCreateProduct(
				repository,
				store,
				embeddings,
				logger,
				metrics,
			)

			product, err := useCase.Execute(context.Background(), tt.input)

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute() error = %v, want error wrapping %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && tt.wantErrorText == "" && err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
			}
			if tt.wantErrorText != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrorText)) {
				t.Errorf("Execute() error = %v, want text %q", err, tt.wantErrorText)
			}
			if (product != nil) != tt.wantProduct {
				t.Errorf("product present = %v, want %v", product != nil, tt.wantProduct)
			}

			if repository.createCalls != tt.wantCreateCalls {
				t.Errorf("repository Create calls = %d, want %d", repository.createCalls, tt.wantCreateCalls)
			}
			if store.incrementCalls != tt.wantIncrementCalls {
				t.Errorf("capacity increment calls = %d, want %d", store.incrementCalls, tt.wantIncrementCalls)
			}
			if store.decrementCalls != tt.wantDecrementCalls {
				t.Errorf("capacity decrement calls = %d, want %d", store.decrementCalls, tt.wantDecrementCalls)
			}
			if embeddings.dispatchCalls != tt.wantEmbeddingCalls {
				t.Errorf("embedding dispatch calls = %d, want %d", embeddings.dispatchCalls, tt.wantEmbeddingCalls)
			}

			if tt.wantIncrementCalls > 0 && store.incrementStore != "store-123" {
				t.Errorf("increment store ID = %q, want store-123", store.incrementStore)
			}
			if tt.wantDecrementCalls > 0 && store.decrementStore != "store-123" {
				t.Errorf("decrement store ID = %q, want store-123", store.decrementStore)
			}
			if tt.wantProduct && repository.createdProduct != product {
				t.Error("repository did not receive the returned product")
			}
			if tt.wantEmbeddingCalls > 0 && product != nil {
				if embeddings.dispatchedProduct.ProductID != product.ID() {
					t.Error("embedding dispatch used the wrong product ID")
				}
				if embeddings.dispatchedProduct.StoreID != product.StoreID() {
					t.Error("embedding dispatch used the wrong store ID")
				}
			}

			wantEvents := append([]string{}, tt.wantAdditionalEvents...)
			wantEvents = append(wantEvents, tt.wantEvent)
			if len(logger.events) != len(wantEvents) {
				t.Fatalf("logged events = %d, want %d: %+v", len(logger.events), len(wantEvents), logger.events)
			}
			if len(metrics.metrics) != len(wantEvents) {
				t.Fatalf("metrics recorded = %d, want %d", len(metrics.metrics), len(wantEvents))
			}
			for i, wantEvent := range wantEvents {
				if logger.events[i].Event != wantEvent {
					t.Errorf("logged event[%d] = %q, want %q", i, logger.events[i].Event, wantEvent)
				}
				if logger.events[i].Operation != "create_product" {
					t.Errorf("logged operation[%d] = %q, want create_product", i, logger.events[i].Operation)
				}
				if i == 0 && tt.wantFailureCategory != "" && logger.events[i].FailureCategory != tt.wantFailureCategory {
					t.Errorf("failure category = %q, want %q", logger.events[i].FailureCategory, tt.wantFailureCategory)
				}
				if metrics.metrics[i].Labels["event"] != wantEvent {
					t.Errorf("metric event[%d] = %q, want %q", i, metrics.metrics[i].Labels["event"], wantEvent)
				}
			}
		})
	}
}

func TestCreateProductObservabilityFailuresDoNotChangeResult(t *testing.T) {
	logger := &loggerFake{err: errors.New("logger unavailable")}
	metrics := &metricsFake{err: errors.New("metrics unavailable")}
	repository := &productRepositoryFake{}
	store := &storeClientFake{ownerStoreID: "store-123"}
	embeddings := &embeddingDispatcherFake{}

	useCase := appusecases.NewCreateProduct(repository, store, embeddings, logger, metrics)
	product, err := useCase.Execute(context.Background(), appusecases.CreateProductInput{
		OwnerID:     "owner-123",
		Name:        "Keyboard",
		Description: "Mechanical keyboard",
		Price:       mustMoney(t, 250000, "NGN"),
		Quantity:    5,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if product == nil {
		t.Fatal("Execute() returned nil product")
	}
	if repository.createCalls != 1 || store.incrementCalls != 1 || embeddings.dispatchCalls != 1 {
		t.Errorf(
			"unexpected call counts: create=%d increment=%d embedding=%d",
			repository.createCalls,
			store.incrementCalls,
			embeddings.dispatchCalls,
		)
	}
}

func mustMoney(t *testing.T, amount int64, currency string) valueobjects.Money {
	t.Helper()

	money, err := valueobjects.NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("create test money: %v", err)
	}
	return money
}
