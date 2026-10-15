package use_cases_test

import (
	"context"
	"errors"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
)

func TestCountProductsByStore(t *testing.T) {
	tests := []struct {
		name      string
		storeID   string
		count     int64
		repoErr   error
		wantCount int64
		wantErr   bool
	}{
		{name: "counts products", storeID: "store-123", count: 7, wantCount: 7},
		{name: "rejects blank store ID", storeID: "  ", wantErr: true},
		{name: "propagates repository error", storeID: "store-123", repoErr: errors.New("database unavailable"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{countByStore: tt.count, countByStoreErr: tt.repoErr}
			uc := appusecases.NewCountProductsByStore(repository, &loggerFake{}, &metricsFake{})
			got, err := uc.Execute(context.Background(), tt.storeID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.wantCount {
				t.Errorf("Execute() = %d, want %d", got, tt.wantCount)
			}
			if tt.storeID == "  " && repository.countByStoreCalls != 0 {
				t.Errorf("CountByStore calls = %d, want 0", repository.countByStoreCalls)
			}
		})
	}
}
