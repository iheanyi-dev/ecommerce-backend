package use_cases_test

import (
	"bytes"
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
)

func TestChangeProductVideoUploadsAndAttachesVideo(t *testing.T) {
	product := newTestProduct(t)
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductVideo(repository, store, storage, &loggerFake{}, &metricsFake{})
	content := []byte("video bytes")

	got, err := useCase.Execute(context.Background(), appusecases.ChangeProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
		ContentType: "video/mp4", Content: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != product || repository.updateCalls != 1 || storage.putCalls != 1 {
		t.Fatalf("product=%v, updates=%d, uploads=%d", got == product, repository.updateCalls, storage.putCalls)
	}
	if !bytes.Equal(storage.putContent, content) {
		t.Error("uploaded video bytes differ from input")
	}
	if storage.putType != "video/mp4" || path.Ext(storage.putReference) != ".mp4" {
		t.Errorf("uploaded type/reference = %q / %q", storage.putType, storage.putReference)
	}
	if video := got.Video(); video == nil || video.StorageReference != storage.putReference {
		t.Errorf("product video = %+v, want uploaded reference", video)
	}
}

func TestChangeProductVideoReplacesExistingVideo(t *testing.T) {
	product := newTestProduct(t)
	if err := product.SetVideo("products/old.mov"); err != nil {
		t.Fatal(err)
	}
	repository := &productRepositoryFake{product: product}
	store := &storeClientFake{ownerStoreID: product.StoreID(), active: true}
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductVideo(repository, store, storage, &loggerFake{}, &metricsFake{})

	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
		ContentType: "video/webm", Content: strings.NewReader("replacement"),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if storage.deleteCalls != 1 || storage.deletedRefs[0] != "products/old.mov" {
		t.Fatalf("deleted refs = %v, want old video", storage.deletedRefs)
	}
	if path.Ext(storage.putReference) != ".webm" {
		t.Errorf("reference extension = %q, want .webm", path.Ext(storage.putReference))
	}
}

func TestChangeProductVideoRejectsUnsupportedContentType(t *testing.T) {
	product := newTestProduct(t)
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductVideo(
		&productRepositoryFake{product: product},
		&storeClientFake{ownerStoreID: product.StoreID(), active: true},
		storage, &loggerFake{}, &metricsFake{},
	)
	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
		ContentType: "video/avi", Content: strings.NewReader("video"),
	})
	if !errors.Is(err, appusecases.ErrInvalidProductVideo) {
		t.Fatalf("Execute() error = %v, want ErrInvalidProductVideo", err)
	}
	if storage.putCalls != 0 {
		t.Errorf("upload calls = %d, want 0", storage.putCalls)
	}
}

func TestChangeProductVideoRestoresPreviousVideoWhenPersistenceFails(t *testing.T) {
	product := newTestProduct(t)
	if err := product.SetVideo("products/original.mp4"); err != nil {
		t.Fatal(err)
	}
	persistenceErr := errors.New("database unavailable")
	storage := &storageClientFake{}
	useCase := appusecases.NewChangeProductVideo(
		&productRepositoryFake{product: product, updateErr: persistenceErr},
		&storeClientFake{ownerStoreID: product.StoreID(), active: true},
		storage, &loggerFake{}, &metricsFake{},
	)
	_, err := useCase.Execute(context.Background(), appusecases.ChangeProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
		ContentType: "video/mp4", Content: strings.NewReader("replacement"),
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want persistence error", err)
	}
	if video := product.Video(); video == nil || video.StorageReference != "products/original.mp4" {
		t.Errorf("video after rollback = %+v", video)
	}
	if storage.deleteCalls != 1 || storage.deletedRefs[0] != storage.putReference {
		t.Errorf("cleanup after rollback = %v, want newly uploaded reference", storage.deletedRefs)
	}
}

func TestRemoveProductVideoRemovesReferenceAndStorageObject(t *testing.T) {
	product := newTestProduct(t)
	if err := product.SetVideo("products/demo.mp4"); err != nil {
		t.Fatal(err)
	}
	repository := &productRepositoryFake{product: product}
	storage := &storageClientFake{}
	useCase := appusecases.NewRemoveProductVideo(
		repository,
		&storeClientFake{ownerStoreID: product.StoreID(), active: true},
		storage, &loggerFake{}, &metricsFake{},
	)
	err := useCase.Execute(context.Background(), appusecases.RemoveProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.updateCalls != 1 || product.Video() != nil {
		t.Errorf("updates=%d, video=%+v; want one update and no video", repository.updateCalls, product.Video())
	}
	if storage.deleteCalls != 1 || storage.deletedRefs[0] != "products/demo.mp4" {
		t.Errorf("deleted refs = %v", storage.deletedRefs)
	}
}

func TestRemoveProductVideoRestoresReferenceWhenPersistenceFails(t *testing.T) {
	product := newTestProduct(t)
	if err := product.SetVideo("products/demo.mp4"); err != nil {
		t.Fatal(err)
	}
	persistenceErr := errors.New("database unavailable")
	storage := &storageClientFake{}
	useCase := appusecases.NewRemoveProductVideo(
		&productRepositoryFake{product: product, updateErr: persistenceErr},
		&storeClientFake{ownerStoreID: product.StoreID(), active: true},
		storage, &loggerFake{}, &metricsFake{},
	)
	err := useCase.Execute(context.Background(), appusecases.RemoveProductVideoInput{
		OwnerID: "owner-123", ProductID: product.ID(),
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want persistence error", err)
	}
	if video := product.Video(); video == nil || video.StorageReference != "products/demo.mp4" {
		t.Errorf("video after rollback = %+v", video)
	}
	if storage.deleteCalls != 0 {
		t.Errorf("storage delete calls = %d, want 0", storage.deleteCalls)
	}
}
