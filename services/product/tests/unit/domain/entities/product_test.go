package entities_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func newValidProduct(t *testing.T, quantity int64) *entities.Product {
	t.Helper()

	price, err := valueobjects.NewMoney(1500000, "NGN")
	if err != nil {
		t.Fatalf("create price: %v", err)
	}

	product, err := entities.NewProduct(
		"store-123",
		"Wireless Headphones",
		"Over-ear wireless headphones",
		price,
		quantity,
	)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	return product
}

func TestNewProductRejectsInvalidData(t *testing.T) {
	price, err := valueobjects.NewMoney(1500000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		storeID     string
		productName string
		description string
		quantity    int64
	}{
		{name: "missing store ID", storeID: "", productName: "Headphones", description: "Wireless", quantity: 1},
		{name: "missing name", storeID: "store-123", productName: "", description: "Wireless", quantity: 1},
		{name: "missing description", storeID: "store-123", productName: "Headphones", description: "", quantity: 1},
		{name: "negative quantity", storeID: "store-123", productName: "Headphones", description: "Wireless", quantity: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewProduct(
				tt.storeID,
				tt.productName,
				tt.description,
				price,
				tt.quantity,
			)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewProductAllowsZeroQuantity(t *testing.T) {
	product := newValidProduct(t, 0)

	if got := product.Quantity(); got != 0 {
		t.Fatalf("quantity = %d, want 0", got)
	}
}

func TestProductSupportsAtMostFourImages(t *testing.T) {
	product := newValidProduct(t, 10)

	for i := 1; i <= 4; i++ {
		ref := "products/product-123/image-" + string(rune('0'+i)) + ".webp"
		if err := product.AddImage(ref); err != nil {
			t.Fatalf("AddImage() #%d: %v", i, err)
		}
	}

	if err := product.AddImage("products/product-123/image-5.webp"); err == nil {
		t.Fatal("expected adding a fifth image to fail")
	}

	if got := len(product.Images()); got != 4 {
		t.Fatalf("image count = %d, want 4", got)
	}
}

func TestProductRejectsDuplicateImageReferences(t *testing.T) {
	product := newValidProduct(t, 10)
	ref := "products/product-123/front.webp"

	if err := product.AddImage(ref); err != nil {
		t.Fatal(err)
	}

	if err := product.AddImage(ref); err == nil {
		t.Fatal("expected duplicate image reference to fail")
	}
}

func TestProductRemovesImageAndFreesSlot(t *testing.T) {
	product := newValidProduct(t, 10)

	for i := 1; i <= 4; i++ {
		ref := "products/product-123/image-" + string(rune('0'+i)) + ".webp"
		if err := product.AddImage(ref); err != nil {
			t.Fatal(err)
		}
	}

	if err := product.RemoveImage("products/product-123/image-2.webp"); err != nil {
		t.Fatalf("RemoveImage(): %v", err)
	}

	if err := product.AddImage("products/product-123/image-5.webp"); err != nil {
		t.Fatalf("adding image after removal should succeed: %v", err)
	}

	if got := len(product.Images()); got != 4 {
		t.Fatalf("image count = %d, want 4", got)
	}
}
