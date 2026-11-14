package use_cases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	application "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

func newLifecyclePromotion(
	t *testing.T,
	startsAt, endsAt time.Time,
) *entities.Promotion {
	t.Helper()

	percentage := int64(1500)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatalf("create discount: %v", err)
	}

	promotion, err := entities.NewPromotion(
		uuid.NewString(),
		nil,
		nil,
		discount,
		startsAt,
		endsAt,
	)
	if err != nil {
		t.Fatalf("create promotion: %v", err)
	}

	return promotion
}

func TestPromotionLifecycleUseCases(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		eventType  string
		wantStatus entities.PromotionStatus
		promotion  func() *entities.Promotion
		execute    func(*promotionRepositoryStub, *promotionOutboxRepositoryStub, *unitOfWorkStub) (dtos.PromotionDTO, error)
	}{
		{
			name:       "cancel promotion",
			eventType:  "promotion.cancelled",
			wantStatus: entities.PromotionCancelled,
			promotion: func() *entities.Promotion {
				return newLifecyclePromotion(t, now.Add(-time.Hour), now.Add(time.Hour))
			},
			execute: func(r *promotionRepositoryStub, o *promotionOutboxRepositoryStub, u *unitOfWorkStub) (dtos.PromotionDTO, error) {
				return application.NewCancelPromotionUseCase(r, o, u).Execute(
					context.Background(),
					dtos.CancelPromotionInput{PromotionID: r.promotion.ID()},
				)
			},
		},
		{
			name:       "activate promotion",
			eventType:  "promotion.activated",
			wantStatus: entities.PromotionActive,
			promotion: func() *entities.Promotion {
				return newLifecyclePromotion(t, now.Add(-time.Hour), now.Add(time.Hour))
			},
			execute: func(r *promotionRepositoryStub, o *promotionOutboxRepositoryStub, u *unitOfWorkStub) (dtos.PromotionDTO, error) {
				return application.NewActivatePromotionUseCase(r, o, u).Execute(
					context.Background(),
					dtos.ActivatePromotionInput{PromotionID: r.promotion.ID()},
				)
			},
		},
		{
			name:       "expire promotion",
			eventType:  "promotion.expired",
			wantStatus: entities.PromotionExpired,
			promotion: func() *entities.Promotion {
				return newLifecyclePromotion(t, now.Add(-2*time.Hour), now.Add(-time.Hour))
			},
			execute: func(r *promotionRepositoryStub, o *promotionOutboxRepositoryStub, u *unitOfWorkStub) (dtos.PromotionDTO, error) {
				return application.NewExpirePromotionUseCase(r, o, u).Execute(
					context.Background(),
					dtos.ExpirePromotionInput{PromotionID: r.promotion.ID()},
				)
			},
		},
		{
			name:       "renew promotion and retain its ID",
			eventType:  "promotion.renewed",
			wantStatus: entities.PromotionScheduled,
			promotion: func() *entities.Promotion {
				return newLifecyclePromotion(t, now.Add(-2*time.Hour), now.Add(-time.Hour))
			},
			execute: func(r *promotionRepositoryStub, o *promotionOutboxRepositoryStub, u *unitOfWorkStub) (dtos.PromotionDTO, error) {
				return application.NewRenewPromotionUseCase(r, o, u).Execute(
					context.Background(),
					dtos.RenewPromotionInput{
						PromotionID: r.promotion.ID(),
						StartsAt:    now.Add(time.Hour),
						EndsAt:      now.Add(24 * time.Hour),
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promotion := tt.promotion()
			originalID := promotion.ID()

			repository := &promotionRepositoryStub{promotion: promotion}
			outbox := &promotionOutboxRepositoryStub{}
			uow := &unitOfWorkStub{}

			result, err := tt.execute(repository, outbox, uow)
			if err != nil {
				t.Fatalf("execute lifecycle: %v", err)
			}

			if result.ID != originalID {
				t.Fatalf("promotion ID changed: got %s, want %s", result.ID, originalID)
			}
			if result.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", result.Status, tt.wantStatus)
			}
			if repository.getCalls != 1 || repository.updateCalls != 1 {
				t.Fatalf("expected one read and update; got reads=%d updates=%d",
					repository.getCalls, repository.updateCalls)
			}
			if uow.calls != 1 || outbox.calls != 1 {
				t.Fatalf("expected one transaction and outbox write; got transactions=%d outbox=%d",
					uow.calls, outbox.calls)
			}
			if outbox.eventType != tt.eventType {
				t.Fatalf("event type = %s, want %s", outbox.eventType, tt.eventType)
			}
			if outbox.promotionID != originalID {
				t.Fatalf("outbox promotion ID = %s, want %s", outbox.promotionID, originalID)
			}
			if _, err := uuid.Parse(outbox.eventID); err != nil {
				t.Fatalf("invalid event UUID: %v", err)
			}

			var payload map[string]json.RawMessage
			if err := json.Unmarshal(outbox.payload, &payload); err != nil {
				t.Fatalf("decode event payload: %v", err)
			}
			if len(payload) != 4 {
				t.Fatalf("expected exactly four event fields, got %d", len(payload))
			}

			var event application.PromotionLifecycleEvent
			if err := json.Unmarshal(outbox.payload, &event); err != nil {
				t.Fatalf("decode lifecycle event: %v", err)
			}
			if event.PromoID != originalID {
				t.Fatalf("event promo ID = %s, want %s", event.PromoID, originalID)
			}
			if event.StoreID != promotion.StoreID() {
				t.Fatalf("event store ID = %s, want %s", event.StoreID, promotion.StoreID())
			}
			if event.Status != string(tt.wantStatus) {
				t.Fatalf("event status = %s, want %s", event.Status, tt.wantStatus)
			}
		})
	}
}

func TestDeletePromotionUseCase(t *testing.T) {
	now := time.Now().UTC()

	t.Run("deletes expired promotion without publishing event", func(t *testing.T) {
		promotion := newLifecyclePromotion(t, now.Add(-2*time.Hour), now.Add(-time.Hour))
		repository := &promotionRepositoryStub{promotion: promotion}
		uow := &unitOfWorkStub{}

		err := application.NewDeletePromotionUseCase(repository, uow).Execute(
			context.Background(),
			dtos.DeletePromotionInput{PromotionID: promotion.ID()},
		)
		if err != nil {
			t.Fatalf("delete expired promotion: %v", err)
		}
		if repository.deleteCalls != 1 || repository.deletedID != promotion.ID() {
			t.Fatalf("expected promotion to be deleted; calls=%d ID=%s",
				repository.deleteCalls, repository.deletedID)
		}
		if uow.calls != 1 {
			t.Fatalf("expected one transaction, got %d", uow.calls)
		}
	})

	t.Run("deletes cancelled promotion", func(t *testing.T) {
		promotion := newLifecyclePromotion(t, now.Add(-time.Hour), now.Add(time.Hour))
		if err := promotion.Cancel(now); err != nil {
			t.Fatalf("cancel promotion: %v", err)
		}

		repository := &promotionRepositoryStub{promotion: promotion}
		err := application.NewDeletePromotionUseCase(repository, &unitOfWorkStub{}).Execute(
			context.Background(),
			dtos.DeletePromotionInput{PromotionID: promotion.ID()},
		)
		if err != nil {
			t.Fatalf("delete cancelled promotion: %v", err)
		}
		if repository.deleteCalls != 1 {
			t.Fatalf("expected one delete, got %d", repository.deleteCalls)
		}
	})

	t.Run("rejects active promotion", func(t *testing.T) {
		promotion := newLifecyclePromotion(t, now.Add(-time.Hour), now.Add(time.Hour))
		repository := &promotionRepositoryStub{promotion: promotion}

		err := application.NewDeletePromotionUseCase(repository, &unitOfWorkStub{}).Execute(
			context.Background(),
			dtos.DeletePromotionInput{PromotionID: promotion.ID()},
		)
		if !errors.Is(err, apperrors.ErrPromotionNotDeletable) {
			t.Fatalf("expected not-deletable error, got %v", err)
		}
		if repository.deleteCalls != 0 {
			t.Fatalf("must not delete active promotion")
		}
	})
}

func TestPromotionLifecycleUseCase_OutboxFailure(t *testing.T) {
	now := time.Now().UTC()
	promotion := newLifecyclePromotion(t, now.Add(-time.Hour), now.Add(time.Hour))
	outboxErr := errors.New("outbox insert failed")

	repository := &promotionRepositoryStub{promotion: promotion}
	outbox := &promotionOutboxRepositoryStub{err: outboxErr}
	uow := &unitOfWorkStub{}

	_, err := application.NewCancelPromotionUseCase(repository, outbox, uow).Execute(
		context.Background(),
		dtos.CancelPromotionInput{PromotionID: promotion.ID()},
	)
	if !errors.Is(err, outboxErr) {
		t.Fatalf("expected outbox error, got %v", err)
	}
	if repository.updateCalls != 1 || outbox.calls != 1 || uow.calls != 1 {
		t.Fatalf("expected update and outbox write inside transaction; updates=%d outbox=%d transactions=%d",
			repository.updateCalls, outbox.calls, uow.calls)
	}
}
