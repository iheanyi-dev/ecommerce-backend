package use_cases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
)

func TestDeleteProductDeletesAndCleansUpResources(t *testing.T) {
	product := newTestProduct(t)
	for _, reference := range []string{"products/image-1.png", "products/image-2.jpg"} {
		if err := product.AddImage(reference); err != nil {
			t.Fatal(err)
		}
	}

	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID()}
	storage := &storageClientFake{}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewDeleteProduct(repository, store, storage, embedding, &loggerFake{}, &metricsFake{})

	err := useCase.Execute(context.Background(), appusecases.DeleteProductInput{
		OwnerID:   "owner-123",
		ProductID: product.ID(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.deleteCalls != 1 || repository.deletedID != product.ID() {
		t.Errorf("repository Delete calls/id = %d/%q, want 1/%q", repository.deleteCalls, repository.deletedID, product.ID())
	}
	if store.decrementCalls != 1 || store.decrementStore != product.StoreID() {
		t.Errorf("store decrement calls/store = %d/%q, want 1/%q", store.decrementCalls, store.decrementStore, product.StoreID())
	}
	if !reflect.DeepEqual(storage.deletedRefs, []string{"products/image-1.png", "products/image-2.jpg"}) {
		t.Errorf("deleted storage references = %v, want both product images", storage.deletedRefs)
	}
	if embedding.deletionCalls != 1 {
		t.Errorf("embedding deletion calls = %d, want 1", embedding.deletionCalls)
	}
	if embedding.deletedProductID != product.ID() || embedding.deletedStoreID != product.StoreID() {
		t.Errorf("embedding deletion target = %q/%q, want %q/%q",
			embedding.deletedProductID, embedding.deletedStoreID, product.ID(), product.StoreID())
	}
}

func TestDeleteProductRejectsWrongOwner(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: "another-store"}
	storage := &storageClientFake{}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewDeleteProduct(repository, store, storage, embedding, &loggerFake{}, &metricsFake{})

	err := useCase.Execute(context.Background(), appusecases.DeleteProductInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if !errors.Is(err, appusecases.ErrProductNotFound) {
		t.Fatalf("Execute() error = %v, want ErrProductNotFound", err)
	}
	if repository.deleteCalls != 0 || store.decrementCalls != 0 {
		t.Errorf("delete/decrement calls = %d/%d, want 0/0", repository.deleteCalls, store.decrementCalls)
	}
	if len(storage.deletedRefs) != 0 || embedding.deletionCalls != 0 {
		t.Errorf("cleanup occurred for unauthorized product")
	}
}

func TestDeleteProductStopsWhenCountDecrementFails(t *testing.T) {
	product := newTestProduct(t)
	decrementErr := errors.New("store service unavailable")
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), decrementErr: decrementErr}
	storage := &storageClientFake{}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewDeleteProduct(repository, store, storage, embedding, &loggerFake{}, &metricsFake{})

	err := useCase.Execute(context.Background(), appusecases.DeleteProductInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if !errors.Is(err, decrementErr) {
		t.Fatalf("Execute() error = %v, want decrement error", err)
	}
	if repository.deleteCalls != 0 {
		t.Errorf("repository Delete calls = %d, want 0", repository.deleteCalls)
	}
	if len(storage.deletedRefs) != 0 || embedding.deletionCalls != 0 {
		t.Errorf("cleanup occurred after count decrement failed")
	}
}

func TestDeleteProductDoesNotCleanUpWhenPersistenceFails(t *testing.T) {
	product := newTestProduct(t)
	if err := product.AddImage("products/image.png"); err != nil {
		t.Fatal(err)
	}
	persistenceErr := errors.New("database unavailable")
	repository := &productRepositoryFake{product: product, deleteErr: persistenceErr}
	store := &storeClientFake{ownerStoreID: product.StoreID()}
	storage := &storageClientFake{}
	embedding := &embeddingDispatcherFake{}
	useCase := appusecases.NewDeleteProduct(repository, store, storage, embedding, &loggerFake{}, &metricsFake{})

	err := useCase.Execute(context.Background(), appusecases.DeleteProductInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want persistence error", err)
	}
	if len(storage.deletedRefs) != 0 || embedding.deletionCalls != 0 {
		t.Errorf("cleanup occurred before successful persistence")
	}
}

func TestDeleteProductCanDeleteFromInactiveStore(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: false}
	useCase := appusecases.NewDeleteProduct(
		repository, store, &storageClientFake{}, &embeddingDispatcherFake{},
		&loggerFake{}, &metricsFake{},
	)

	err := useCase.Execute(context.Background(), appusecases.DeleteProductInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want deletion to be allowed for inactive store", err)
	}
	if repository.deleteCalls != 1 {
		t.Errorf("repository Delete calls = %d, want 1", repository.deleteCalls)
	}
}
