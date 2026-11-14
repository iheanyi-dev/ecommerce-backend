package use_cases_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	application "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
)

type promotionRepositoryStub struct {
	promotion   *entities.Promotion
	now         time.Time
	err         error
	getErr      error
	updateErr   error
	deleteErr   error
	getCalls    int
	updateCalls int
	deleteCalls int
	deletedID   string
	calls       int
}

func (r *promotionRepositoryStub) CreateIfNoUnexpiredConflict(
	_ context.Context,
	promotion *entities.Promotion,
	now time.Time,
) error {
	r.calls++
	r.promotion = promotion
	r.now = now
	return r.err
}

func (r *promotionRepositoryStub) ListByStore(
	_ context.Context, _ string, _, _ int,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func (r *promotionRepositoryStub) ListActiveForTarget(
	_ context.Context, _, _, _ string, _ time.Time,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func (r *promotionRepositoryStub) ListAvailableForCustomer(
	_ context.Context, _ string, _ time.Time, _, _ int,
) ([]*entities.Promotion, error) {
	return nil, nil
}

func (r *promotionRepositoryStub) GetByID(
	_ context.Context, _ string,
) (*entities.Promotion, error) {
	r.getCalls++
	return r.promotion, r.getErr
}

func (r *promotionRepositoryStub) Update(
	_ context.Context, promotion *entities.Promotion,
) error {
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	r.promotion = promotion
	return nil
}

func (r *promotionRepositoryStub) Delete(
	_ context.Context, promotionID string,
) error {
	r.deleteCalls++
	r.deletedID = promotionID
	return r.deleteErr
}

type promotionOutboxRepositoryStub struct {
	eventID     string
	promotionID string
	eventType   string
	payload     []byte
	err         error
	calls       int
}

func (r *promotionOutboxRepositoryStub) Create(
	_ context.Context,
	eventID string,
	promotionID string,
	eventType string,
	payload []byte,
) error {
	r.calls++
	r.eventID = eventID
	r.promotionID = promotionID
	r.eventType = eventType
	r.payload = append([]byte(nil), payload...)
	return r.err
}

func (r *promotionOutboxRepositoryStub) ClaimPending(
	_ context.Context,
	_ int,
	_ time.Duration,
) ([]ports.PendingPromotionEvent, error) {
	return nil, nil
}

func (r *promotionOutboxRepositoryStub) MarkPublished(
	_ context.Context,
	_ string,
	_ string,
) error {
	return nil
}

func (r *promotionOutboxRepositoryStub) RecordFailure(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) error {
	return nil
}

type unitOfWorkStub struct {
	calls int
	err   error
}

func (u *unitOfWorkStub) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	u.calls++
	if u.err != nil {
		return u.err
	}
	return fn(ctx)
}

func TestCreatePromotionUseCase_Execute(t *testing.T) {
	now := time.Now().UTC()
	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()
	percentage := int64(1500)

	tests := []struct {
		name    string
		input   dtos.CreatePromotionInput
		repoErr error
		wantErr string
	}{
		{
			name: "creates store-level promotion",
			input: dtos.CreatePromotionInput{
				StoreID:               storeID,
				PercentageBasisPoints: &percentage,
				StartsAt:              now.Add(time.Hour),
				EndsAt:                now.Add(24 * time.Hour),
			},
		},
		{
			name: "creates product and customer targeted promotion",
			input: dtos.CreatePromotionInput{
				StoreID:               storeID,
				ProductID:             &productID,
				CustomerID:            &customerID,
				PercentageBasisPoints: &percentage,
				StartsAt:              now.Add(time.Hour),
				EndsAt:                now.Add(24 * time.Hour),
			},
		},
		{
			name: "creates promotion with fixed discount",
			input: dtos.CreatePromotionInput{
				StoreID: storeID,
				FixedAmount: &dtos.MoneyDTO{
					Amount:   50000,
					Currency: "NGN",
				},
				StartsAt: now.Add(time.Hour),
				EndsAt:   now.Add(24 * time.Hour),
			},
		},
		{
			name: "rejects invalid store ID",
			input: dtos.CreatePromotionInput{
				StoreID:               "invalid",
				PercentageBasisPoints: &percentage,
				StartsAt:              now.Add(time.Hour),
				EndsAt:                now.Add(24 * time.Hour),
			},
			wantErr: "invalid promotion",
		},
		{
			name: "rejects missing discount",
			input: dtos.CreatePromotionInput{
				StoreID:  storeID,
				StartsAt: now.Add(time.Hour),
				EndsAt:   now.Add(24 * time.Hour),
			},
			wantErr: "invalid discount",
		},
		{
			name: "returns targeting conflict",
			input: dtos.CreatePromotionInput{
				StoreID:               storeID,
				PercentageBasisPoints: &percentage,
				StartsAt:              now.Add(time.Hour),
				EndsAt:                now.Add(24 * time.Hour),
			},
			repoErr: apperrors.ErrPromotionConflict,
			wantErr: "an unexpired promotion already exists for this targeting scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &promotionRepositoryStub{err: tt.repoErr}
			outbox := &promotionOutboxRepositoryStub{}
			uow := &unitOfWorkStub{}

			useCase := application.NewCreatePromotionUseCase(
				repository,
				outbox,
				uow,
			)

			result, err := useCase.Execute(context.Background(), tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				if repository.calls != 0 && tt.repoErr == nil {
					t.Fatal("repository should not be called for invalid input")
				}
				if outbox.calls != 0 {
					t.Fatal("outbox should not be called when promotion creation fails")
				}
				if uow.calls != 0 && tt.repoErr == nil {
					t.Fatal("transaction should not start for invalid input")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if uow.calls != 1 {
				t.Fatalf("expected one transaction, got %d", uow.calls)
			}
			if repository.calls != 1 {
				t.Fatalf("expected one repository call, got %d", repository.calls)
			}
			if outbox.calls != 1 {
				t.Fatalf("expected one outbox call, got %d", outbox.calls)
			}
			if repository.promotion == nil {
				t.Fatal("expected promotion to be persisted")
			}
			if result.ID == "" {
				t.Fatal("expected generated promotion ID")
			}
			if result.StoreID != tt.input.StoreID {
				t.Fatalf("expected store ID %s, got %s", tt.input.StoreID, result.StoreID)
			}
			if result.Status != entities.PromotionScheduled {
				t.Fatalf("expected scheduled status, got %s", result.Status)
			}
			if repository.now.IsZero() || repository.now.Location() != time.UTC {
				t.Fatal("expected repository to receive current UTC time")
			}
			if outbox.eventID == "" {
				t.Fatal("expected generated outbox event ID")
			}
			if _, err := uuid.Parse(outbox.eventID); err != nil {
				t.Fatalf("expected UUID event ID, got %q", outbox.eventID)
			}
			if outbox.promotionID != result.ID {
				t.Fatalf("expected outbox promotion ID %s, got %s", result.ID, outbox.promotionID)
			}
			if outbox.eventType != "promotion.created" {
				t.Fatalf("unexpected event type: %s", outbox.eventType)
			}

			var event application.PromotionCreatedEvent
			if err := json.Unmarshal(outbox.payload, &event); err != nil {
				t.Fatalf("decode outbox payload: %v", err)
			}
			if event.PromoID != result.ID {
				t.Fatalf("expected event promo ID %s, got %s", result.ID, event.PromoID)
			}
			if event.StoreID != result.StoreID {
				t.Fatalf("expected event store ID %s, got %s", result.StoreID, event.StoreID)
			}
			if event.Status != string(result.Status) {
				t.Fatalf("expected event status %s, got %s", result.Status, event.Status)
			}
			if (event.ProductID == nil) != (tt.input.ProductID == nil) {
				t.Fatal("event product ID presence does not match input")
			}
			if event.ProductID != nil && *event.ProductID != *tt.input.ProductID {
				t.Fatalf("expected event product ID %s, got %s", *tt.input.ProductID, *event.ProductID)
			}
		})
	}
}

func TestCreatePromotionUseCase_Execute_OutboxFailure(t *testing.T) {
	now := time.Now().UTC()
	percentage := int64(1500)
	outboxErr := errors.New("outbox insert failed")

	repository := &promotionRepositoryStub{}
	outbox := &promotionOutboxRepositoryStub{err: outboxErr}
	uow := &unitOfWorkStub{}

	useCase := application.NewCreatePromotionUseCase(repository, outbox, uow)

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{
		StoreID:               uuid.NewString(),
		PercentageBasisPoints: &percentage,
		StartsAt:              now.Add(time.Hour),
		EndsAt:                now.Add(24 * time.Hour),
	})

	if !errors.Is(err, outboxErr) {
		t.Fatalf("expected outbox error, got %v", err)
	}
	if repository.calls != 1 || outbox.calls != 1 || uow.calls != 1 {
		t.Fatalf("expected promotion and outbox writes inside one transaction; got repo=%d outbox=%d uow=%d",
			repository.calls, outbox.calls, uow.calls)
	}
}

func TestCreatePromotionUseCase_Execute_RepositoryRequired(t *testing.T) {
	var useCase *application.CreatePromotionUseCase

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{})

	if !errors.Is(err, apperrors.ErrPromotionRepositoryRequired) {
		t.Fatalf("expected repository required error, got %v", err)
	}
}

func TestCreatePromotionUseCase_Execute_OutboxRepositoryRequired(t *testing.T) {
	useCase := application.NewCreatePromotionUseCase(
		&promotionRepositoryStub{},
		nil,
		&unitOfWorkStub{},
	)

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{})

	if !errors.Is(err, apperrors.ErrPromotionOutboxRepositoryRequired) {
		t.Fatalf("expected outbox repository required error, got %v", err)
	}
}

func TestCreatePromotionUseCase_Execute_UnitOfWorkRequired(t *testing.T) {
	useCase := application.NewCreatePromotionUseCase(
		&promotionRepositoryStub{},
		&promotionOutboxRepositoryStub{},
		nil,
	)

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{})

	if !errors.Is(err, apperrors.ErrPromotionUnitOfWorkRequired) {
		t.Fatalf("expected unit of work required error, got %v", err)
	}
}

func TestCreatePromotionUseCase_Execute_RepositoryFailure(t *testing.T) {
	now := time.Now().UTC()
	percentage := int64(1500)
	repoErr := errors.New("promotion persistence failed")

	repository := &promotionRepositoryStub{err: repoErr}
	outbox := &promotionOutboxRepositoryStub{}
	uow := &unitOfWorkStub{}

	useCase := application.NewCreatePromotionUseCase(repository, outbox, uow)

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{
		StoreID:               uuid.NewString(),
		PercentageBasisPoints: &percentage,
		StartsAt:              now.Add(time.Hour),
		EndsAt:                now.Add(24 * time.Hour),
	})

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if uow.calls != 1 {
		t.Fatalf("expected one transaction, got %d", uow.calls)
	}
	if repository.calls != 1 {
		t.Fatalf("expected one repository call, got %d", repository.calls)
	}
	if outbox.calls != 0 {
		t.Fatalf("outbox must not be called after repository failure, got %d calls", outbox.calls)
	}
}

func TestCreatePromotionUseCase_Execute_TransactionFailure(t *testing.T) {
	now := time.Now().UTC()
	percentage := int64(1500)
	transactionErr := errors.New("transaction failed")

	repository := &promotionRepositoryStub{}
	outbox := &promotionOutboxRepositoryStub{}
	uow := &unitOfWorkStub{err: transactionErr}

	useCase := application.NewCreatePromotionUseCase(repository, outbox, uow)

	_, err := useCase.Execute(context.Background(), dtos.CreatePromotionInput{
		StoreID:               uuid.NewString(),
		PercentageBasisPoints: &percentage,
		StartsAt:              now.Add(time.Hour),
		EndsAt:                now.Add(24 * time.Hour),
	})

	if !errors.Is(err, transactionErr) {
		t.Fatalf("expected transaction error, got %v", err)
	}
	if uow.calls != 1 {
		t.Fatalf("expected one transaction, got %d", uow.calls)
	}
	if repository.calls != 0 {
		t.Fatalf("repository must not be called when transaction fails, got %d calls", repository.calls)
	}
	if outbox.calls != 0 {
		t.Fatalf("outbox must not be called when transaction fails, got %d calls", outbox.calls)
	}
}
