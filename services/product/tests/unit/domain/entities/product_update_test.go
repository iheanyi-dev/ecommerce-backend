package entities_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func TestProductHasGeneratedIDAndTimestamps(t *testing.T) {
	product := newValidProduct(t, 5)

	if product.ID() == "" {
		t.Fatal("product ID must not be empty")
	}
	if product.CreatedAt().IsZero() || product.UpdatedAt().IsZero() {
		t.Fatal("product timestamps must be initialized")
	}
	if product.UpdatedAt().Before(product.CreatedAt()) {
		t.Fatal("updated timestamp must not precede creation timestamp")
	}
}

func TestUpdateDetailsRejectsEmptyValues(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.UpdateDetails("", "Description"); err == nil {
		t.Fatal("expected empty name to be rejected")
	}
	if err := product.UpdateDetails("Name", " "); err == nil {
		t.Fatal("expected empty description to be rejected")
	}
}

func TestUpdateDetailsTrimsValues(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.UpdateDetails("  Updated Name  ", "  Updated description  "); err != nil {
		t.Fatal(err)
	}
	if product.Name() != "Updated Name" {
		t.Fatalf("name = %q, want trimmed name", product.Name())
	}
	if product.Description() != "Updated description" {
		t.Fatalf("description = %q, want trimmed description", product.Description())
	}
}

func TestUpdateQuantityRejectsNegativeAndAllowsZero(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.UpdateQuantity(-1); err == nil {
		t.Fatal("expected negative quantity to be rejected")
	}
	if err := product.UpdateQuantity(0); err != nil {
		t.Fatalf("zero quantity should be allowed: %v", err)
	}
	if product.Quantity() != 0 {
		t.Fatalf("quantity = %d, want 0", product.Quantity())
	}
}

func TestUpdatePrice(t *testing.T) {
	product := newValidProduct(t, 5)
	price, err := valueobjects.NewMoney(2500000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	product.UpdatePrice(price)

	if !product.Price().Equal(price) {
		t.Fatal("product price was not updated")
	}
}

func TestReorderImages(t *testing.T) {
	product := newValidProduct(t, 5)
	refs := []string{
		"products/product-123/front.webp",
		"products/product-123/side.webp",
		"products/product-123/back.webp",
	}

	for _, ref := range refs {
		if err := product.AddImage(ref); err != nil {
			t.Fatal(err)
		}
	}

	if err := product.ReorderImages([]string{refs[2], refs[0], refs[1]}); err != nil {
		t.Fatalf("ReorderImages(): %v", err)
	}

	got := product.Images()
	want := []string{refs[2], refs[0], refs[1]}
	for i := range want {
		if got[i].StorageReference != want[i] {
			t.Fatalf("image at position %d = %q, want %q", i, got[i].StorageReference, want[i])
		}
	}
}

func TestReorderImagesRejectsIncompleteOrder(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.AddImage("products/product-123/front.webp"); err != nil {
		t.Fatal(err)
	}
	if err := product.AddImage("products/product-123/back.webp"); err != nil {
		t.Fatal(err)
	}

	if err := product.ReorderImages([]string{"products/product-123/front.webp"}); err == nil {
		t.Fatal("expected incomplete image order to be rejected")
	}
}

func TestImagesReturnsCopy(t *testing.T) {
	product := newValidProduct(t, 5)
	if err := product.AddImage("products/product-123/front.webp"); err != nil {
		t.Fatal(err)
	}

	images := product.Images()
	images[0].StorageReference = "tampered"

	if product.Images()[0].StorageReference != "products/product-123/front.webp" {
		t.Fatal("caller mutated aggregate image state")
	}
}

// Keep the compile-time reference explicit: Product remains the domain aggregate.
var _ *entities.Product
