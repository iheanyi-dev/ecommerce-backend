package use_cases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

type getPromotionByIDRepositoryStub struct {
	promotion *entities.Promotion
	err       error
	calls     int
}

func (r *getPromotionByIDRepositoryStub) GetByID(
	_ context.Context, _ string,
) (*entities.Promotion, error) {
	r.calls++
	return r.promotion, r.err
}

func (*getPromotionByIDRepositoryStub) CreateIfNoUnexpiredConflict(
	context.Context, *entities.Promotion, time.Time,
) error {
	return nil
}

func (*getPromotionByIDRepositoryStub) Update(
	context.Context, *entities.Promotion,
) error {
	return nil
}

func (*getPromotionByIDRepositoryStub) Delete(context.Context, string) error {
	return nil
}

func (*getPromotionByIDRepositoryStub) ListByStore(
	context.Context, string, int, int,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func (*getPromotionByIDRepositoryStub) ListActiveForTarget(
	context.Context, string, string, string, time.Time,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func (*getPromotionByIDRepositoryStub) ListAvailableForCustomer(
	context.Context, string, time.Time, int, int,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func TestGetPromotionByIDUseCase_Execute(t *testing.T) {
	now := time.Now().UTC()
	storeID := uuid.NewString()
	percentage := int64(1000)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatal(err)
	}
	promotion, err := entities.NewPromotion(
		storeID, nil, nil, discount,
		now.Add(-time.Hour), now.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		id        string
		promotion *entities.Promotion
		repoErr   error
		wantErr   error
		wantCalls int
	}{
		{
			name:      "retrieves promotion",
			id:        promotion.ID(),
			promotion: promotion,
			wantCalls: 1,
		},
		{
			name:      "rejects invalid ID",
			id:        "invalid",
			wantErr:   apperrors.ErrInvalidPromotionQuery,
			wantCalls: 0,
		},
		{
			name:      "returns not found",
			id:        uuid.NewString(),
			wantErr:   apperrors.ErrPromotionNotFound,
			wantCalls: 1,
		},
		{
			name:      "propagates repository error",
			id:        uuid.NewString(),
			repoErr:   errors.New("database unavailable"),
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &getPromotionByIDRepositoryStub{
				promotion: tt.promotion,
				err:       tt.repoErr,
			}
			uc := appusecases.NewGetPromotionByIDUseCase(repository)

			result, err := uc.Execute(context.Background(), dtos.GetPromotionByIDInput{
				PromotionID: tt.id,
			})

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.repoErr != nil && !errors.Is(err, tt.repoErr) {
				t.Fatalf("error = %v, want repository error", err)
			}
			if tt.wantErr == nil && tt.repoErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repository.calls != tt.wantCalls {
				t.Fatalf("repository calls = %d, want %d", repository.calls, tt.wantCalls)
			}
			if tt.name == "retrieves promotion" && result.ID != promotion.ID() {
				t.Fatalf("promotion ID = %q, want %q", result.ID, promotion.ID())
			}
		})
	}
}
