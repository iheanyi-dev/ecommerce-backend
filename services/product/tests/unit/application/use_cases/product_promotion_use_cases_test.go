package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

type productPromotionRepositoryFake struct {
	added              *entities.ProductPromotion
	updated            *entities.ProductPromotion
	removedStoreID     string
	removedPromotionID string
	err                error
}

func (f *productPromotionRepositoryFake) Add(
	_ context.Context,
	promotion *entities.ProductPromotion,
) error {
	f.added = promotion
	return f.err
}

func (f *productPromotionRepositoryFake) Update(
	_ context.Context,
	promotion *entities.ProductPromotion,
) error {
	f.updated = promotion
	return f.err
}

func (f *productPromotionRepositoryFake) Remove(
	_ context.Context,
	storeID, promotionID string,
) error {
	f.removedStoreID = storeID
	f.removedPromotionID = promotionID
	return f.err
}

func (f *productPromotionRepositoryFake) Exists(
	_ context.Context,
	_ string,
	_ *string,
) (bool, error) {
	return false, f.err
}

var _ ports.ProductPromotionRepository = (*productPromotionRepositoryFake)(nil)

func TestAddProductPromotion(t *testing.T) {
	repository := &productPromotionRepositoryFake{}
	uc := use_cases.NewAddProductPromotion(repository, nil, nil)

	err := uc.Execute(context.Background(), use_cases.AddProductPromotionInput{
		StoreID: "store-1", PromotionID: "promotion-1", Status: "upcoming",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.added == nil {
		t.Fatal("expected promotion to be added")
	}
	if repository.added.Status() != "upcoming" {
		t.Fatalf("status = %q, want upcoming", repository.added.Status())
	}
	if repository.added.ProductID() != nil {
		t.Fatal("expected store-level promotion to have nil product ID")
	}
}

func TestUpdateProductPromotion(t *testing.T) {
	repository := &productPromotionRepositoryFake{}
	uc := use_cases.NewUpdateProductPromotion(repository, nil, nil)
	productID := "product-1"

	err := uc.Execute(context.Background(), use_cases.UpdateProductPromotionInput{
		StoreID: "store-1", ProductID: &productID,
		PromotionID: "promotion-1", Status: "active",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.updated == nil {
		t.Fatal("expected promotion to be updated")
	}
	if repository.updated.Status() != "active" {
		t.Fatalf("status = %q, want active", repository.updated.Status())
	}
	if got := repository.updated.ProductID(); got == nil || *got != productID {
		t.Fatalf("product ID = %v, want %q", got, productID)
	}
}

func TestRemoveProductPromotion(t *testing.T) {
	repository := &productPromotionRepositoryFake{}
	uc := use_cases.NewRemoveProductPromotion(repository, nil, nil)

	err := uc.Execute(context.Background(), use_cases.RemoveProductPromotionInput{
		StoreID: " store-1 ", PromotionID: " promotion-1 ",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.removedStoreID != "store-1" {
		t.Fatalf("store ID = %q, want store-1", repository.removedStoreID)
	}
	if repository.removedPromotionID != "promotion-1" {
		t.Fatalf("promotion ID = %q, want promotion-1", repository.removedPromotionID)
	}
}

func TestProductPromotionUseCasesPropagateRepositoryErrors(t *testing.T) {
	repositoryError := errors.New("database unavailable")

	tests := []struct {
		name string
		run  func(*productPromotionRepositoryFake) error
	}{
		{
			name: "add",
			run: func(repository *productPromotionRepositoryFake) error {
				return use_cases.NewAddProductPromotion(repository, nil, nil).Execute(
					context.Background(),
					use_cases.AddProductPromotionInput{
						StoreID: "store-1", PromotionID: "promotion-1", Status: "active",
					},
				)
			},
		},
		{
			name: "update",
			run: func(repository *productPromotionRepositoryFake) error {
				return use_cases.NewUpdateProductPromotion(repository, nil, nil).Execute(
					context.Background(),
					use_cases.UpdateProductPromotionInput{
						StoreID: "store-1", PromotionID: "promotion-1", Status: "active",
					},
				)
			},
		},
		{
			name: "remove",
			run: func(repository *productPromotionRepositoryFake) error {
				return use_cases.NewRemoveProductPromotion(repository, nil, nil).Execute(
					context.Background(),
					use_cases.RemoveProductPromotionInput{
						StoreID: "store-1", PromotionID: "promotion-1",
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productPromotionRepositoryFake{err: repositoryError}
			err := tt.run(repository)
			if !errors.Is(err, repositoryError) {
				t.Fatalf("error = %v, want wrapped repository error", err)
			}
		})
	}
}

func TestAddProductPromotionRejectsInvalidInput(t *testing.T) {
	repository := &productPromotionRepositoryFake{}
	uc := use_cases.NewAddProductPromotion(repository, nil, nil)

	err := uc.Execute(context.Background(), use_cases.AddProductPromotionInput{
		StoreID: "store-1", PromotionID: "", Status: "active",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if repository.added != nil {
		t.Fatal("repository should not be called for invalid input")
	}
}
