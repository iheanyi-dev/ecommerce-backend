package use_cases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

func TestGetProductsByIDs(t *testing.T) {
	first := newTestProduct(t)
	second := newTestProduct(t)
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name               string
		ids                []string
		repositoryResult   []*entities.Product
		repositoryErr      error
		wantErr            error
		wantIDs            []string
		wantRepositoryCall int
		wantEvent          string
		wantFailure        string
	}{
		{
			name:               "returns products in requested order",
			ids:                []string{second.ID(), first.ID()},
			repositoryResult:   []*entities.Product{first, second},
			wantIDs:            []string{second.ID(), first.ID()},
			wantRepositoryCall: 1,
			wantEvent:          "product.get_by_ids.succeeded",
		},
		{
			name:               "preserves repeated requested IDs",
			ids:                []string{first.ID(), first.ID()},
			repositoryResult:   []*entities.Product{first},
			wantIDs:            []string{first.ID(), first.ID()},
			wantRepositoryCall: 1,
			wantEvent:          "product.get_by_ids.succeeded",
		},
		{
			name:               "rejects empty IDs",
			ids:                []string{},
			wantErr:            appusecases.ErrInvalidProductIDs,
			wantRepositoryCall: 0,
			wantEvent:          "product.get_by_ids.validation_failed",
			wantFailure:        "validation",
		},
		{
			name:               "rejects blank ID",
			ids:                []string{first.ID(), "  "},
			wantErr:            appusecases.ErrInvalidProductIDs,
			wantRepositoryCall: 0,
			wantEvent:          "product.get_by_ids.validation_failed",
			wantFailure:        "validation",
		},
		{
			name:               "fails when a requested product is missing",
			ids:                []string{first.ID(), second.ID()},
			repositoryResult:   []*entities.Product{first},
			wantErr:            appusecases.ErrProductNotFound,
			wantRepositoryCall: 1,
			wantEvent:          "product.get_by_ids.not_found",
		},
		{
			name:               "propagates repository failure",
			ids:                []string{first.ID()},
			repositoryErr:      repositoryErr,
			wantErr:            repositoryErr,
			wantRepositoryCall: 1,
			wantEvent:          "product.get_by_ids.repository_failed",
			wantFailure:        "persistence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &productRepositoryFake{
				productsByIDs: tt.repositoryResult,
				getByIDsErr:   tt.repositoryErr,
			}
			logger := &loggerFake{}
			metrics := &metricsFake{}
			useCase := appusecases.NewGetProductsByIDs(repository, logger, metrics, nil)

			got, err := useCase.Execute(context.Background(), tt.ids)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Execute() unexpected error: %v", err)
			}

			if tt.wantErr != nil && got != nil {
				t.Errorf("products = %v, want nil on failure", got)
			}
			if tt.wantErr == nil {
				gotIDs := make([]string, 0, len(got))
				for _, product := range got {
					gotIDs = append(gotIDs, product.ID())
				}
				if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
					t.Errorf("returned IDs = %v, want %v", gotIDs, tt.wantIDs)
				}
			}

			if repository.getByIDsCalls != tt.wantRepositoryCall {
				t.Errorf("bulk repository calls = %d, want %d", repository.getByIDsCalls, tt.wantRepositoryCall)
			}
			if tt.wantRepositoryCall > 0 && !reflect.DeepEqual(repository.gotIDs, tt.ids) {
				t.Errorf("repository received IDs = %v, want %v", repository.gotIDs, tt.ids)
			}

			if len(logger.events) != 1 {
				t.Fatalf("logged events = %d, want 1", len(logger.events))
			}
			if logger.events[0].Event != tt.wantEvent {
				t.Errorf("logged event = %q, want %q", logger.events[0].Event, tt.wantEvent)
			}
			if logger.events[0].Operation != "get_products_by_ids" {
				t.Errorf("logged operation = %q", logger.events[0].Operation)
			}
			if logger.events[0].FailureCategory != tt.wantFailure {
				t.Errorf("failure category = %q, want %q", logger.events[0].FailureCategory, tt.wantFailure)
			}

			if len(metrics.metrics) != 1 {
				t.Fatalf("metrics recorded = %d, want 1", len(metrics.metrics))
			}
			metric := metrics.metrics[0]
			if metric.Name != "product_operations_total" || metric.Value != 1 {
				t.Errorf("unexpected metric: %+v", metric)
			}
			if metric.Labels["operation"] != "get_products_by_ids" ||
				metric.Labels["event"] != tt.wantEvent {
				t.Errorf("unexpected metric labels: %+v", metric.Labels)
			}
		})
	}
}
