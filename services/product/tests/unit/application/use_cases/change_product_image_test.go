package use_cases_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"path"
	"strings"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
)

type storageClientFake struct {
	putErr       error
	deleteErr    error
	putCalls     int
	deleteCalls  int
	putReference string
	putType      string
	putContent   []byte
	deletedRefs  []string
}

func (f *storageClientFake) PutObject(
	_ context.Context,
	reference, contentType string,
	content io.Reader,
) error {
	f.putCalls++
	f.putReference = reference
	f.putType = contentType
	f.putContent, _ = io.ReadAll(content)
	return f.putErr
}

func (f *storageClientFake) DeleteObject(_ context.Context, reference string) error {
	f.deleteCalls++
	f.deletedRefs = append(f.deletedRefs, reference)
	return f.deleteErr
}

func TestChangeProductImageAddsImage(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	logger := &loggerFake{}
	metrics := &metricsFake{}

	useCase := appusecases.NewChangeProductImage(repository, store, storage, logger, metrics)
	content := []byte("fake image bytes")

	got, err := useCase.Execute(context.Background(), appusecases.ChangeProductImageInput{
		OwnerID:     "owner-123",
		ProductID:   product.ID(),
		ContentType: "image/png",
		Content:     bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != product {
		t.Fatal("Execute() did not return the updated product")
	}
	if repository.updateCalls != 1 {
		t.Fatalf("repository Update calls = %d, want 1", repository.updateCalls)
	}
	if storage.putCalls != 1 {
		t.Fatalf("storage PutObject calls = %d, want 1", storage.putCalls)
	}
	if storage.putType != "image/png" {
		t.Errorf("uploaded content type = %q, want image/png", storage.putType)
	}
	if !bytes.Equal(storage.putContent, content) {
		t.Error("uploaded image bytes differ from input")
	}
	if !strings.HasPrefix(storage.putReference, "products/"+product.ID()+"/") {
		t.Errorf("storage reference %q does not use product-scoped prefix", storage.putReference)
	}
	if path.Ext(storage.putReference) != ".png" {
		t.Errorf("storage reference extension = %q, want .png", path.Ext(storage.putReference))
	}
	if _, params, err := mime.ParseMediaType(storage.putType); err != nil || len(params) != 0 {
		t.Errorf("unexpected uploaded media type %q", storage.putType)
	}
	images := got.Images()
	if len(images) != 1 || images[0].StorageReference != storage.putReference {
		t.Errorf("product images = %+v, want uploaded reference", images)
	}
	for _, reference := range storage.deletedRefs {
		if reference == "products/old.png" {
			t.Error("old image was deleted when product persistence failed")
		}
	}
	if store.gotOwnerID != "owner-123" {
		t.Errorf("owner lookup ID = %q, want owner-123", store.gotOwnerID)
	}
}

func TestChangeProductImageReplacesImageAfterUploadAndPersistence(t *testing.T) {
	product := newTestProduct(t)
	if err := product.AddImage("products/old.png"); err != nil {
		t.Fatal(err)
	}

	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductImage(repository, store, storage, &loggerFake{}, &metricsFake{})

	got, err := useCase.Execute(context.Background(), appusecases.ChangeProductImageInput{
		OwnerID:                  "owner-123",
		ProductID:                product.ID(),
		ExistingStorageReference: "products/old.png",
		ContentType:              "image/jpeg",
		Content:                  bytes.NewReader([]byte("replacement")),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != product {
		t.Fatal("Execute() did not return the updated product")
	}
	if storage.putCalls != 1 || repository.updateCalls != 1 {
		t.Fatalf("upload calls = %d, update calls = %d; want 1 each", storage.putCalls, repository.updateCalls)
	}
	if storage.deleteCalls != 1 || storage.deletedRefs[0] != "products/old.png" {
		t.Fatalf("deleted refs = %v, want only the old reference", storage.deletedRefs)
	}
	images := got.Images()
	if len(images) != 1 || images[0].StorageReference != storage.putReference {
		t.Fatalf("product images = %+v, want replacement reference", images)
	}
	if path.Ext(storage.putReference) != ".jpg" {
		t.Errorf("storage reference extension = %q, want .jpg", path.Ext(storage.putReference))
	}
}

func TestChangeProductImageRejectsInvalidContentTypeBeforeDependencies(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductImage(repository, store, storage, &loggerFake{}, &metricsFake{})

	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductImageInput{
		OwnerID:     "owner-123",
		ProductID:   product.ID(),
		ContentType: "text/plain",
		Content:     strings.NewReader("not an image"),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid content type error")
	}
	if storage.putCalls != 0 || repository.updateCalls != 0 {
		t.Errorf("upload calls = %d, update calls = %d; want both 0", storage.putCalls, repository.updateCalls)
	}
}

func TestChangeProductImageDoesNotPersistWhenUploadFails(t *testing.T) {
	product := newTestProduct(t)
	uploadErr := errors.New("storage unavailable")
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{putErr: uploadErr}
	useCase := appusecases.NewChangeProductImage(repository, store, storage, &loggerFake{}, &metricsFake{})

	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductImageInput{
		OwnerID:     "owner-123",
		ProductID:   product.ID(),
		ContentType: "image/webp",
		Content:     strings.NewReader("image"),
	})
	if !errors.Is(err, uploadErr) {
		t.Fatalf("Execute() error = %v, want upload error", err)
	}
	if repository.updateCalls != 0 {
		t.Errorf("repository Update calls = %d, want 0", repository.updateCalls)
	}
	if len(product.Images()) != 0 {
		t.Errorf("product images = %+v, want unchanged", product.Images())
	}
}

func TestChangeProductImageDoesNotDeleteOldImageWhenPersistenceFails(t *testing.T) {
	product := newTestProduct(t)
	if err := product.AddImage("products/old.png"); err != nil {
		t.Fatal(err)
	}
	persistenceErr := errors.New("database unavailable")
	repository := &productRepositoryFake{product: product, updateErr: persistenceErr}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductImage(repository, store, storage, &loggerFake{}, &metricsFake{})

	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductImageInput{
		OwnerID:                  "owner-123",
		ProductID:                product.ID(),
		ExistingStorageReference: "products/old.png",
		ContentType:              "image/png",
		Content:                  strings.NewReader("replacement"),
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want persistence error", err)
	}
	for _, reference := range storage.deletedRefs {
		if reference == "products/old.png" {
			t.Error("old image was deleted when product persistence failed")
		}
	}
	if len(product.Images()) != 1 || product.Images()[0].StorageReference != "products/old.png" {
		t.Errorf("in-memory product images = %+v, want original reference", product.Images())
	}
}
