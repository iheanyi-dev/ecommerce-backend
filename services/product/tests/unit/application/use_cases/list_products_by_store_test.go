package use_cases_test

import (
	"context"
	"errors"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestListProductsByStore(t *testing.T) {
	product := newTestProduct(t)
	tests := []struct {
		name      string
		storeID   string
		active    bool
		checkErr  error
		listErr   error
		wantCount int
		wantErr   bool
		wantList  bool
	}{
		{name: "lists active store products", storeID: product.StoreID(), active: true, wantCount: 1, wantList: true},
		{name: "inactive store returns empty page", storeID: product.StoreID()},
		{name: "rejects blank store", storeID: " ", wantErr: true},
		{name: "propagates store check error", storeID: product.StoreID(), checkErr: errors.New("store unavailable"), wantErr: true},
		{name: "propagates repository error", storeID: product.StoreID(), active: true, listErr: errors.New("database unavailable"), wantErr: true, wantList: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				products: []*entities.Product{product}, total: 1, listByStoreErr: tt.listErr,
			}
			store := &storeClientFake{active: tt.active, checkErr: tt.checkErr}
			uc := appusecases.NewListProductsByStore(repository, store, &loggerFake{}, &metricsFake{}, nil)
			page, err := uc.Execute(context.Background(), tt.storeID, 10, 0)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(page.Products) != tt.wantCount {
				t.Errorf("product count = %d, want %d", len(page.Products), tt.wantCount)
			}
			if tt.wantList && repository.listByStoreCalls != 1 {
				t.Errorf("ListByStore calls = %d, want 1", repository.listByStoreCalls)
			}
			if !tt.wantList && repository.listByStoreCalls != 0 {
				t.Errorf("ListByStore calls = %d, want 0", repository.listByStoreCalls)
			}
		})
	}
}
