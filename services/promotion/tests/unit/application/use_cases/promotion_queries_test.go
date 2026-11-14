package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

type promotionRepositoryQueryStub struct {
	byStore       []*entities.Promotion
	forTarget     []*entities.Promotion
	forCustomer   []*entities.Promotion
	err           error
	gotStoreID    string
	gotProductID  string
	gotCustomerID string
	gotLimit      int
	gotOffset     int
	gotNow        time.Time
}

func (s *promotionRepositoryQueryStub) CreateIfNoUnexpiredConflict(
	_ context.Context,
	_ *entities.Promotion,
	_ time.Time,
) error {
	return s.err
}

func (s *promotionRepositoryQueryStub) ListByStore(
	_ context.Context,
	storeID string,
	limit, offset int,
) ([]*entities.Promotion, error) {
	s.gotStoreID = storeID
	s.gotLimit = limit
	s.gotOffset = offset
	return s.byStore, s.err
}

func (s *promotionRepositoryQueryStub) ListActiveForTarget(
	_ context.Context,
	storeID, productID, customerID string,
	now time.Time,
) ([]*entities.Promotion, error) {
	s.gotStoreID = storeID
	s.gotProductID = productID
	s.gotCustomerID = customerID
	s.gotNow = now
	return s.forTarget, s.err
}

func (s *promotionRepositoryQueryStub) ListAvailableForCustomer(
	_ context.Context,
	customerID string,
	now time.Time,
	limit, offset int,
) ([]*entities.Promotion, error) {
	s.gotCustomerID = customerID
	s.gotNow = now
	s.gotLimit = limit
	s.gotOffset = offset
	return s.forCustomer, s.err
}

func (s *promotionRepositoryQueryStub) GetByID(
	_ context.Context, _ string,
) (*entities.Promotion, error) {
	return nil, nil
}

func (s *promotionRepositoryQueryStub) Update(
	_ context.Context, _ *entities.Promotion,
) error {
	return nil
}

func (s *promotionRepositoryQueryStub) Delete(
	_ context.Context, _ string,
) error {
	return nil
}

func newPromotion(
	t *testing.T,
	storeID string,
	productID, customerID *string,
	startsAt, endsAt time.Time,
) *entities.Promotion {
	t.Helper()

	percentage := int64(1000)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatalf("create discount: %v", err)
	}

	promotion, err := entities.NewPromotion(
		storeID,
		productID,
		customerID,
		discount,
		startsAt,
		endsAt,
	)
	if err != nil {
		t.Fatalf("create promotion: %v", err)
	}

	return promotion
}

func stringPointer(value string) *string {
	return &value
}

func TestGetPromosByStore(t *testing.T) {
	storeID := uuid.NewString()
	now := time.Now().UTC()

	active := newPromotion(
		t, storeID, nil, nil,
		now.Add(-time.Hour), now.Add(time.Hour),
	)
	expired := newPromotion(
		t, storeID, nil, nil,
		now.Add(-2*time.Hour), now.Add(-time.Hour),
	)

	reader := &promotionRepositoryQueryStub{
		byStore: []*entities.Promotion{active, expired},
	}
	uc := use_cases.NewGetPromosByStoreUseCase(reader)

	result, err := uc.Execute(context.Background(), dtos.GetPromosByStoreInput{
		StoreID: storeID,
		Limit:   20,
		Offset:  40,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 promotions, got %d", len(result))
	}
	if result[0].ID != active.ID() || result[1].ID != expired.ID() {
		t.Fatal("returned promotions do not match repository results")
	}
	if reader.gotStoreID != storeID || reader.gotLimit != 20 || reader.gotOffset != 40 {
		t.Fatal("repository did not receive the expected store and pagination arguments")
	}
}

func TestGetPromosByStoreValidation(t *testing.T) {
	uc := use_cases.NewGetPromosByStoreUseCase(&promotionRepositoryQueryStub{})

	tests := []struct {
		name  string
		input dtos.GetPromosByStoreInput
	}{
		{
			name: "invalid store ID",
			input: dtos.GetPromosByStoreInput{
				StoreID: "invalid",
				Limit:   10,
			},
		},
		{
			name: "zero limit",
			input: dtos.GetPromosByStoreInput{
				StoreID: uuid.NewString(),
				Limit:   0,
			},
		},
		{
			name: "negative offset",
			input: dtos.GetPromosByStoreInput{
				StoreID: uuid.NewString(),
				Limit:   10,
				Offset:  -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), tt.input)
			if !errors.Is(err, apperrors.ErrInvalidPromotionQuery) {
				t.Fatalf("expected ErrInvalidPromotionQuery, got %v", err)
			}
		})
	}
}

func TestGetPromosByStoreReaderError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	uc := use_cases.NewGetPromosByStoreUseCase(&promotionRepositoryQueryStub{err: expectedErr})

	_, err := uc.Execute(context.Background(), dtos.GetPromosByStoreInput{
		StoreID: uuid.NewString(),
		Limit:   10,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped repository error, got %v", err)
	}
}

func TestGetPromoForUserSelectsHighestPriority(t *testing.T) {
	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()
	now := time.Now().UTC()

	storeWide := newPromotion(t, storeID, nil, nil,
		now.Add(-time.Hour), now.Add(time.Hour))
	customerOnly := newPromotion(t, storeID, nil, &customerID,
		now.Add(-time.Hour), now.Add(time.Hour))
	productOnly := newPromotion(t, storeID, &productID, nil,
		now.Add(-time.Hour), now.Add(time.Hour))
	productAndCustomer := newPromotion(t, storeID, &productID, &customerID,
		now.Add(-time.Hour), now.Add(time.Hour))

	reader := &promotionRepositoryQueryStub{
		forTarget: []*entities.Promotion{
			storeWide,
			customerOnly,
			productOnly,
			productAndCustomer,
		},
	}
	uc := use_cases.NewGetPromoForUserUseCase(reader)

	result, err := uc.Execute(context.Background(), dtos.GetPromoForUserInput{
		StoreID:    storeID,
		ProductID:  productID,
		CustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ID != productAndCustomer.ID() {
		t.Fatalf("expected highest-priority promotion %s, got %s",
			productAndCustomer.ID(), result.ID)
	}
	if reader.gotStoreID != storeID ||
		reader.gotProductID != productID ||
		reader.gotCustomerID != customerID {
		t.Fatal("repository did not receive the expected targeting arguments")
	}
	if reader.gotNow.IsZero() {
		t.Fatal("repository did not receive the current time")
	}
}

func TestGetPromoForUserFallsBackToNextPriority(t *testing.T) {
	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()
	now := time.Now().UTC()

	productOnly := newPromotion(t, storeID, &productID, nil,
		now.Add(-time.Hour), now.Add(time.Hour))
	storeWide := newPromotion(t, storeID, nil, nil,
		now.Add(-time.Hour), now.Add(time.Hour))

	reader := &promotionRepositoryQueryStub{
		forTarget: []*entities.Promotion{storeWide, productOnly},
	}
	uc := use_cases.NewGetPromoForUserUseCase(reader)

	result, err := uc.Execute(context.Background(), dtos.GetPromoForUserInput{
		StoreID:    storeID,
		ProductID:  productID,
		CustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.ID != productOnly.ID() {
		t.Fatalf("expected product promotion %s, got %s", productOnly.ID(), result.ID)
	}
}

func TestGetPromoForUserNotFound(t *testing.T) {
	uc := use_cases.NewGetPromoForUserUseCase(&promotionRepositoryQueryStub{})

	_, err := uc.Execute(context.Background(), dtos.GetPromoForUserInput{
		StoreID:    uuid.NewString(),
		ProductID:  uuid.NewString(),
		CustomerID: uuid.NewString(),
	})
	if !errors.Is(err, apperrors.ErrPromotionNotFound) {
		t.Fatalf("expected ErrPromotionNotFound, got %v", err)
	}
}

func TestGetPromoForUserValidation(t *testing.T) {
	uc := use_cases.NewGetPromoForUserUseCase(&promotionRepositoryQueryStub{})

	_, err := uc.Execute(context.Background(), dtos.GetPromoForUserInput{
		StoreID:    "invalid",
		ProductID:  uuid.NewString(),
		CustomerID: uuid.NewString(),
	})
	if !errors.Is(err, apperrors.ErrInvalidPromotionQuery) {
		t.Fatalf("expected ErrInvalidPromotionQuery, got %v", err)
	}
}

func TestGetPromoForUserReaderError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	uc := use_cases.NewGetPromoForUserUseCase(&promotionRepositoryQueryStub{err: expectedErr})

	_, err := uc.Execute(context.Background(), dtos.GetPromoForUserInput{
		StoreID:    uuid.NewString(),
		ProductID:  uuid.NewString(),
		CustomerID: uuid.NewString(),
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped repository error, got %v", err)
	}
}

func TestGetPromosForUser(t *testing.T) {
	storeID := uuid.NewString()
	customerID := uuid.NewString()
	productID := uuid.NewString()
	now := time.Now().UTC()

	storeWide := newPromotion(t, storeID, nil, nil,
		now.Add(-time.Hour), now.Add(time.Hour))
	negotiated := newPromotion(t, storeID, &productID, &customerID,
		now.Add(-time.Hour), now.Add(time.Hour))

	reader := &promotionRepositoryQueryStub{
		forCustomer: []*entities.Promotion{storeWide, negotiated},
	}
	uc := use_cases.NewGetPromosForUserUseCase(reader)

	result, err := uc.Execute(context.Background(), dtos.GetPromosForUserInput{
		CustomerID: customerID,
		Limit:      15,
		Offset:     30,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 promotions, got %d", len(result))
	}
	if result[0].ID != storeWide.ID() || result[1].ID != negotiated.ID() {
		t.Fatal("returned promotions do not match repository results")
	}
	if reader.gotCustomerID != customerID ||
		reader.gotLimit != 15 ||
		reader.gotOffset != 30 {
		t.Fatal("repository did not receive the expected customer and pagination arguments")
	}
	if reader.gotNow.IsZero() {
		t.Fatal("repository did not receive the current time")
	}
}

func TestGetPromosForUserValidation(t *testing.T) {
	uc := use_cases.NewGetPromosForUserUseCase(&promotionRepositoryQueryStub{})

	_, err := uc.Execute(context.Background(), dtos.GetPromosForUserInput{
		CustomerID: "invalid",
		Limit:      10,
	})
	if !errors.Is(err, apperrors.ErrInvalidPromotionQuery) {
		t.Fatalf("expected ErrInvalidPromotionQuery, got %v", err)
	}

	_, err = uc.Execute(context.Background(), dtos.GetPromosForUserInput{
		CustomerID: uuid.NewString(),
		Limit:      10,
		Offset:     -1,
	})
	if !errors.Is(err, apperrors.ErrInvalidPromotionQuery) {
		t.Fatalf("expected ErrInvalidPromotionQuery for negative offset, got %v", err)
	}
}

func TestGetPromosForUserReaderError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	uc := use_cases.NewGetPromosForUserUseCase(&promotionRepositoryQueryStub{err: expectedErr})

	_, err := uc.Execute(context.Background(), dtos.GetPromosForUserInput{
		CustomerID: uuid.NewString(),
		Limit:      10,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped repository error, got %v", err)
	}
}
