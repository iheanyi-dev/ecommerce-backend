package entities_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestNewProductPromotion(t *testing.T) {
	productID := " product-123 "

	promotion, err := entities.NewProductPromotion(
		" store-123 ", &productID, " promotion-123 ", " active ",
	)
	if err != nil {
		t.Fatal(err)
	}

	if promotion.StoreID() != "store-123" {
		t.Fatal("store ID was not normalized")
	}
	if promotion.PromotionID() != "promotion-123" {
		t.Fatal("promotion ID was not normalized")
	}
	if promotion.ProductID() == nil || *promotion.ProductID() != "product-123" {
		t.Fatal("product ID was not normalized")
	}
	if promotion.Status() != "active" {
		t.Fatal("status was not preserved")
	}
}

func TestNewStoreLevelPromotion(t *testing.T) {
	promotion, err := entities.NewProductPromotion(
		"store-123", nil, "promotion-123", "active",
	)
	if err != nil {
		t.Fatal(err)
	}

	if promotion.ProductID() != nil {
		t.Fatal("store-level promotion should have nil product ID")
	}
}

func TestNewProductPromotionRejectsInvalidData(t *testing.T) {
	blank := " "

	tests := []struct {
		name        string
		storeID     string
		productID   *string
		promotionID string
		status      string
	}{
		{"missing store", "", nil, "promo-1", "active"},
		{"missing promotion", "store-1", nil, "", "active"},
		{"missing status", "store-1", nil, "promo-1", ""},
		{"blank product ID", "store-1", &blank, "promo-1", "active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewProductPromotion(
				tt.storeID, tt.productID, tt.promotionID, tt.status,
			)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestProductPromotionProductIDDoesNotExposeInternalPointer(t *testing.T) {
	id := "product-1"
	promotion, err := entities.NewProductPromotion(
		"store-1", &id, "promo-1", "active",
	)
	if err != nil {
		t.Fatal(err)
	}

	got := promotion.ProductID()
	*got = "mutated"

	if *promotion.ProductID() != "product-1" {
		t.Fatal("ProductID exposes internal state")
	}
}
