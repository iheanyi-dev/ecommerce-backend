package use_cases_test

import (
	"context"
	"errors"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestListMyProducts(t *testing.T) {
	product := newTestProduct(t)
	tests := []struct {
		name      string
		ownerID   string
		storeID   string
		lookupErr error
		listErr   error
		wantCount int
		wantTotal int64
		wantErr   bool
		wantList  bool
	}{
		{name: "lists owner products", ownerID: "owner-123", storeID: product.StoreID(), wantCount: 1, wantTotal: 1, wantList: true},
		{name: "no store returns empty page", ownerID: "owner-123"},
		{name: "rejects blank owner", ownerID: " ", wantErr: true},
		{name: "propagates store lookup error", ownerID: "owner-123", lookupErr: errors.New("store unavailable"), wantErr: true},
		{name: "propagates repository error", ownerID: "owner-123", storeID: "store-123", listErr: errors.New("database unavailable"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				products: []*entities.Product{product}, total: 1, listByStoreErr: tt.listErr,
			}
			store := &storeClientFake{ownerStoreID: tt.storeID, ownerLookupErr: tt.lookupErr}
			uc := appusecases.NewListMyProducts(repository, store, &loggerFake{}, &metricsFake{})
			page, err := uc.Execute(context.Background(), tt.ownerID, 10, 0)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(page.Products) != tt.wantCount || page.Total != tt.wantTotal {
				t.Errorf("page count/total = %d/%d, want %d/%d", len(page.Products), page.Total, tt.wantCount, tt.wantTotal)
			}
			if tt.wantList && (repository.listByStoreCalls != 1 || repository.listLimit != 10 || repository.listOffset != 0) {
				t.Errorf("ListByStore calls/limit/offset = %d/%d/%d", repository.listByStoreCalls, repository.listLimit, repository.listOffset)
			}
			if !tt.wantList && repository.listByStoreCalls != 0 {
				t.Errorf("ListByStore calls = %d, want 0", repository.listByStoreCalls)
			}
		})
	}
}
