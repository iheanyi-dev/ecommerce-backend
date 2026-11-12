package use_cases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestGetProductsByIDsForCart(t *testing.T) {
	first := newTestProduct(t)
	second := newTestProduct(t)
	missingID := "missing-product-id"
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name             string
		ids              []string
		repositoryResult []*entities.Product
		repositoryErr    error
		wantErr          error
		wantIDs          []string
		wantCalls        int
	}{
		{
			name:             "returns products in requested order",
			ids:              []string{second.ID(), first.ID()},
			repositoryResult: []*entities.Product{first, second},
			wantIDs:          []string{second.ID(), first.ID()},
			wantCalls:        1,
		},
		{
			name:             "omits missing products",
			ids:              []string{first.ID(), missingID, second.ID()},
			repositoryResult: []*entities.Product{second, first},
			wantIDs:          []string{first.ID(), second.ID()},
			wantCalls:        1,
		},
		{
			name:             "preserves duplicates of existing products",
			ids:              []string{first.ID(), missingID, first.ID()},
			repositoryResult: []*entities.Product{first},
			wantIDs:          []string{first.ID(), first.ID()},
			wantCalls:        1,
		},
		{
			name:      "returns empty result when all products are missing",
			ids:       []string{missingID},
			wantIDs:   []string{},
			wantCalls: 1,
		},
		{
			name:    "rejects empty IDs",
			ids:     []string{},
			wantErr: appusecases.ErrInvalidProductIDs,
		},
		{
			name:    "rejects blank ID",
			ids:     []string{first.ID(), "  "},
			wantErr: appusecases.ErrInvalidProductIDs,
		},
		{
			name:          "propagates repository failure",
			ids:           []string{first.ID()},
			repositoryErr: repositoryErr,
			wantErr:       repositoryErr,
			wantCalls:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				productsByIDs: tt.repositoryResult,
				getByIDsErr:   tt.repositoryErr,
			}
			useCase := appusecases.NewGetProductsByIDsForCart(repository, nil, nil, nil)

			got, err := useCase.Execute(context.Background(), tt.ids)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("products = %v, want nil on failure", got)
				}
			} else {
				if err != nil {
					t.Fatalf("Execute() unexpected error: %v", err)
				}
				gotIDs := make([]string, 0, len(got))
				for _, product := range got {
					gotIDs = append(gotIDs, product.ID())
				}
				if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
					t.Errorf("returned IDs = %v, want %v", gotIDs, tt.wantIDs)
				}
			}

			if repository.getByIDsCalls != tt.wantCalls {
				t.Errorf("repository calls = %d, want %d", repository.getByIDsCalls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && !reflect.DeepEqual(repository.gotIDs, tt.ids) {
				t.Errorf("repository received IDs = %v, want %v", repository.gotIDs, tt.ids)
			}
		})
	}
}
