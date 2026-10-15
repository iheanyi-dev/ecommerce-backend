package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

type searchProductsRepositoryFake struct {
	products []*entities.Product
	total    int64
	err      error
	calls    int
	filters  ports.ProductListFilters
	limit    int
	offset   int
}

func (f *searchProductsRepositoryFake) Create(context.Context, *entities.Product) error {
	return nil
}

func (f *searchProductsRepositoryFake) GetByID(context.Context, string) (*entities.Product, error) {
	return nil, nil
}

func (f *searchProductsRepositoryFake) GetByIDs(context.Context, []string) ([]*entities.Product, error) {
	return nil, nil
}

func (f *searchProductsRepositoryFake) Update(context.Context, *entities.Product) error {
	return nil
}

func (f *searchProductsRepositoryFake) Delete(context.Context, string) error {
	return nil
}

func (f *searchProductsRepositoryFake) ListByStore(context.Context, string, int, int) ([]*entities.Product, int64, error) {
	return nil, 0, nil
}

func (f *searchProductsRepositoryFake) CountByStore(context.Context, string) (int64, error) {
	return 0, nil
}

func (f *searchProductsRepositoryFake) ListProducts(
	_ context.Context,
	filters ports.ProductListFilters,
	limit, offset int,
) ([]*entities.Product, int64, error) {
	f.calls++
	f.filters = filters
	f.limit = limit
	f.offset = offset
	return f.products, f.total, f.err
}

type searchProductsStoreFake struct {
	activeIDs []string
	err       error
	calls     int
	storeIDs  []string
}

func (f *searchProductsStoreFake) GetStoreIDByOwnerID(context.Context, string) (string, error) {
	return "", nil
}

func (f *searchProductsStoreFake) IncrementProductCount(context.Context, string) error {
	return nil
}

func (f *searchProductsStoreFake) DecrementProductCount(context.Context, string) error {
	return nil
}

func (f *searchProductsStoreFake) IsStoreActive(context.Context, string) (bool, error) {
	return false, nil
}

func (f *searchProductsStoreFake) FilterActiveStoreIDs(
	_ context.Context,
	storeIDs []string,
) ([]string, error) {
	f.calls++
	f.storeIDs = append([]string(nil), storeIDs...)
	return f.activeIDs, f.err
}

func newSearchProduct(t *testing.T, storeID, name string) *entities.Product {
	t.Helper()

	price, err := valueobjects.NewMoney(2500, "NGN")
	if err != nil {
		t.Fatalf("create price: %v", err)
	}

	product, err := entities.NewProduct(storeID, name, "Test description", price, 4)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	return product
}

func TestSearchProductsReturnsProductEntitiesAndForwardsFilters(t *testing.T) {
	product := newSearchProduct(t, "store-active", "Wireless Headphones")
	minPrice, err := valueobjects.NewMoney(1000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	maxPrice, err := valueobjects.NewMoney(5000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	repository := &searchProductsRepositoryFake{
		products: []*entities.Product{product},
		total:    17,
	}
	store := &searchProductsStoreFake{activeIDs: []string{"store-active"}}
	search := appusecases.NewSearchProducts(repository, store, nil, nil)

	page, err := search.Execute(context.Background(), appusecases.SearchProductsInput{
		Query:       "  headphones  ",
		MinPrice:    &minPrice,
		MaxPrice:    &maxPrice,
		InStockOnly: true,
		SortBy:      appusecases.SortPriceAsc,
		Limit:       10,
		Offset:      5,
	})
	if err != nil {
		t.Fatalf("SearchProducts() error = %v", err)
	}

	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
	if repository.filters.Query != "headphones" {
		t.Errorf("query = %q, want trimmed query", repository.filters.Query)
	}
	if repository.filters.MinPrice != &minPrice || repository.filters.MaxPrice != &maxPrice {
		t.Error("price filters were not forwarded")
	}
	if !repository.filters.InStockOnly {
		t.Error("InStockOnly filter was not forwarded")
	}
	if repository.filters.SortBy != appusecases.SortPriceAsc {
		t.Errorf("sort = %q, want %q", repository.filters.SortBy, appusecases.SortPriceAsc)
	}
	if repository.limit != 10 || repository.offset != 5 {
		t.Errorf("pagination = (%d, %d), want (10, 5)", repository.limit, repository.offset)
	}
	if page.Total != 17 || page.Limit != 10 || page.Offset != 5 {
		t.Errorf("unexpected page metadata: %+v", page)
	}
	if len(page.Products) != 1 || page.Products[0] != product {
		t.Fatal("SearchProducts must return the original Product entity")
	}
	if store.calls != 1 {
		t.Errorf("active-store filtering calls = %d, want 1", store.calls)
	}
}

func TestSearchProductsExcludesProductsFromInactiveStores(t *testing.T) {
	activeProduct := newSearchProduct(t, "store-active", "Active product")
	inactiveProduct := newSearchProduct(t, "store-inactive", "Inactive product")

	repository := &searchProductsRepositoryFake{
		products: []*entities.Product{activeProduct, inactiveProduct},
		total:    2,
	}
	store := &searchProductsStoreFake{activeIDs: []string{"store-active"}}
	search := appusecases.NewSearchProducts(repository, store, nil, nil)

	page, err := search.Execute(context.Background(), appusecases.SearchProductsInput{Query: "product"})
	if err != nil {
		t.Fatalf("SearchProducts() error = %v", err)
	}
	if len(page.Products) != 1 || page.Products[0] != activeProduct {
		t.Fatalf("products = %v, want only the active-store product", page.Products)
	}
}

func TestSearchProductsRejectsBlankQueryWithoutDependencies(t *testing.T) {
	repository := &searchProductsRepositoryFake{}
	store := &searchProductsStoreFake{}
	search := appusecases.NewSearchProducts(repository, store, nil, nil)

	_, err := search.Execute(context.Background(), appusecases.SearchProductsInput{Query: " \t\n "})
	if !errors.Is(err, appusecases.ErrInvalidMarketplaceProductSearch) {
		t.Fatalf("error = %v, want ErrInvalidMarketplaceProductSearch", err)
	}
	if repository.calls != 0 || store.calls != 0 {
		t.Fatal("blank query must be rejected before calling dependencies")
	}
}

func TestSearchProductsPropagatesRepositoryAndStoreErrors(t *testing.T) {
	t.Run("repository failure", func(t *testing.T) {
		wantErr := errors.New("database unavailable")
		repository := &searchProductsRepositoryFake{err: wantErr}
		store := &searchProductsStoreFake{}
		search := appusecases.NewSearchProducts(repository, store, nil, nil)

		_, err := search.Execute(context.Background(), appusecases.SearchProductsInput{Query: "phone"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want wrapped repository error", err)
		}
		if store.calls != 0 {
			t.Fatal("store filtering should not run after repository failure")
		}
	})

	t.Run("store filtering failure", func(t *testing.T) {
		wantErr := errors.New("store service unavailable")
		repository := &searchProductsRepositoryFake{
			products: []*entities.Product{newSearchProduct(t, "store-1", "Phone")},
			total:    1,
		}
		store := &searchProductsStoreFake{err: wantErr}
		search := appusecases.NewSearchProducts(repository, store, nil, nil)

		_, err := search.Execute(context.Background(), appusecases.SearchProductsInput{Query: "phone"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want wrapped store error", err)
		}
	})
}
