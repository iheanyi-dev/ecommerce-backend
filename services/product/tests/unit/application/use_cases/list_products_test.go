package use_cases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func TestListProductsFiltersInactiveStoresAndForwardsFilters(t *testing.T) {
	activeProduct := newTestProduct(t)
	inactiveProduct, err := entities.NewProduct("inactive-store", "Hidden", "Not public", activeProduct.Price(), 1)
	if err != nil {
		t.Fatal(err)
	}
	repository := &productRepositoryFake{
		products: []*entities.Product{activeProduct, inactiveProduct}, total: 2,
	}
	store := &storeClientFake{activeStoreIDs: []string{activeProduct.StoreID()}}
	uc := appusecases.NewListProducts(repository, store, &loggerFake{}, &metricsFake{})

	minPrice, err := valueobjects.NewMoney(100000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	page, err := uc.Execute(context.Background(), appusecases.ListProductsInput{
		Query: "  shoes  ", MinPrice: &minPrice, InStockOnly: true,
		SortBy: appusecases.SortPriceAsc, Limit: 10, Offset: 5,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(page.Products) != 1 || page.Products[0].ID() != activeProduct.ID() {
		t.Fatalf("products = %+v, want only active-store product", page.Products)
	}
	if page.Total != 2 || page.Limit != 10 || page.Offset != 5 {
		t.Errorf("page metadata = total:%d limit:%d offset:%d", page.Total, page.Limit, page.Offset)
	}
	if repository.listProductsCalls != 1 {
		t.Fatalf("ListProducts calls = %d, want 1", repository.listProductsCalls)
	}
	got := repository.listFilters
	if got.Query != "shoes" || got.SortBy != appusecases.SortPriceAsc || !got.InStockOnly || got.MinPrice == nil || !got.MinPrice.Equal(minPrice) {
		t.Errorf("filters = %+v, want normalized query and supplied filters", got)
	}
	if !reflect.DeepEqual(store.filteredStoreIDs, []string{activeProduct.StoreID(), "inactive-store"}) {
		t.Errorf("filtered store IDs = %v", store.filteredStoreIDs)
	}
}

func TestListProductsRejectsInvalidFilters(t *testing.T) {
	price, err := valueobjects.NewMoney(1000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	higherPrice, err := valueobjects.NewMoney(2000, "USD")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input appusecases.ListProductsInput
	}{
		{name: "unsupported sort", input: appusecases.ListProductsInput{SortBy: "random", Limit: 10}},
		{name: "currency mismatch", input: appusecases.ListProductsInput{MinPrice: &price, MaxPrice: &higherPrice, Limit: 10}},
		{name: "negative offset", input: appusecases.ListProductsInput{Limit: 10, Offset: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{}
			uc := appusecases.NewListProducts(repository, &storeClientFake{}, &loggerFake{}, &metricsFake{})
			if _, err := uc.Execute(context.Background(), tt.input); err == nil {
				t.Fatal("Execute() error = nil, want validation error")
			}
			if repository.listProductsCalls != 0 {
				t.Errorf("ListProducts calls = %d, want 0", repository.listProductsCalls)
			}
		})
	}
}

func TestListProductsPropagatesRepositoryAndStoreFilterErrors(t *testing.T) {
	product := newTestProduct(t)
	repositoryErr := errors.New("database unavailable")
	storeErr := errors.New("store service unavailable")

	t.Run("repository error", func(t *testing.T) {
		repository := &productRepositoryFake{listProductsErr: repositoryErr}
		uc := appusecases.NewListProducts(repository, &storeClientFake{}, &loggerFake{}, &metricsFake{})
		if _, err := uc.Execute(context.Background(), appusecases.ListProductsInput{Limit: 10}); !errors.Is(err, repositoryErr) {
			t.Fatalf("Execute() error = %v, want repository error", err)
		}
	})
	t.Run("store filter error", func(t *testing.T) {
		repository := &productRepositoryFake{products: []*entities.Product{product}, total: 1}
		store := &storeClientFake{filterActiveErr: storeErr}
		uc := appusecases.NewListProducts(repository, store, &loggerFake{}, &metricsFake{})
		if _, err := uc.Execute(context.Background(), appusecases.ListProductsInput{Limit: 10}); !errors.Is(err, storeErr) {
			t.Fatalf("Execute() error = %v, want store filter error", err)
		}
	})
}
