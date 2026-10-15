package use_cases_test

import (
	"context"
	"errors"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func TestUpdateProductUpdatesDetailsAndRefreshesEmbedding(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewUpdateProduct(repository, store, embedding, &loggerFake{}, &metricsFake{})

	name := "Updated product"
	description := "Updated description"
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:     "owner-123",
		ProductID:   product.ID(),
		Name:        &name,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if product.Name() != name || product.Description() != description {
		t.Fatalf("product details = %q / %q, want %q / %q", product.Name(), product.Description(), name, description)
	}
	if repository.updateCalls != 1 {
		t.Fatalf("repository Update calls = %d, want 1", repository.updateCalls)
	}
	if embedding.dispatchCalls != 1 {
		t.Fatalf("embedding dispatches = %d, want 1", embedding.dispatchCalls)
	}
	if embedding.dispatchedProduct.ProductID != product.ID() ||
		embedding.dispatchedProduct.StoreID != product.StoreID() ||
		embedding.dispatchedProduct.Name != name ||
		embedding.dispatchedProduct.Description != description {
		t.Fatalf("embedding payload = %+v, want updated product details", embedding.dispatchedProduct)
	}
}

func TestUpdateProductUpdatesPriceAndQuantityWithoutRefreshingEmbedding(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewUpdateProduct(repository, store, embedding, &loggerFake{}, &metricsFake{})

	price, err := valueobjects.NewMoney(200000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	quantity := int64(0)
	_, err = useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Price:     &price,
		Quantity:  &quantity,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !product.Price().Equal(price) {
		t.Errorf("product price = %+v, want %+v", product.Price(), price)
	}
	if product.Quantity() != 0 {
		t.Errorf("product quantity = %d, want 0", product.Quantity())
	}
	if embedding.dispatchCalls != 0 {
		t.Errorf("embedding dispatches = %d, want 0", embedding.dispatchCalls)
	}
	if repository.updateCalls != 1 {
		t.Errorf("repository Update calls = %d, want 1", repository.updateCalls)
	}
}

func TestUpdateProductRejectsEmptyName(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewUpdateProduct(repository, store, embedding, &loggerFake{}, &metricsFake{})

	name := "   "
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Name:      &name,
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if repository.updateCalls != 0 {
		t.Errorf("repository Update calls = %d, want 0", repository.updateCalls)
	}
}

func TestUpdateProductRejectsNoFields(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	useCase := appusecases.NewUpdateProduct(repository, store, &embeddingDispatcherFake{}, &loggerFake{}, &metricsFake{})

	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if repository.updateCalls != 0 {
		t.Errorf("repository Update calls = %d, want 0", repository.updateCalls)
	}
}

func TestUpdateProductRejectsWrongOwner(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: "another-store", active: true}
	useCase := appusecases.NewUpdateProduct(repository, store, &embeddingDispatcherFake{}, &loggerFake{}, &metricsFake{})

	name := "Changed"
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Name:      &name,
	})
	if !errors.Is(err, appusecases.ErrProductNotFound) {
		t.Fatalf("Execute() error = %v, want ErrProductNotFound", err)
	}
	if repository.updateCalls != 0 {
		t.Errorf("repository Update calls = %d, want 0", repository.updateCalls)
	}
}

func TestUpdateProductRejectsInactiveStore(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: false}
	useCase := appusecases.NewUpdateProduct(repository, store, &embeddingDispatcherFake{}, &loggerFake{}, &metricsFake{})

	name := "Changed"
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Name:      &name,
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want inactive-store error")
	}
	if repository.updateCalls != 0 {
		t.Errorf("repository Update calls = %d, want 0", repository.updateCalls)
	}
}

func TestUpdateProductPersistenceFailureDoesNotDispatchEmbedding(t *testing.T) {
	product := newTestProduct(t)
	persistenceErr := errors.New("database unavailable")
	repository := &productRepositoryFake{product: product, updateErr: persistenceErr}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewUpdateProduct(repository, store, embedding, &loggerFake{}, &metricsFake{})

	name := "Changed"
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Name:      &name,
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want persistence error", err)
	}
	if embedding.dispatchCalls != 0 {
		t.Errorf("embedding dispatches = %d, want 0", embedding.dispatchCalls)
	}
}

func TestUpdateProductDoesNotDispatchEmbeddingForUnchangedDetails(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewUpdateProduct(repository, store, embedding, &loggerFake{}, &metricsFake{})

	name := product.Name()
	_, err := useCase.Execute(context.Background(), appusecases.UpdateProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
		Name:      &name,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if embedding.dispatchCalls != 0 {
		t.Errorf("embedding dispatches = %d, want 0 for unchanged details", embedding.dispatchCalls)
	}
}
