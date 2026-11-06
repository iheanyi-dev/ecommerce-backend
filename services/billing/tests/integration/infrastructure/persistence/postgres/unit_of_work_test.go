package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres"
)

func TestUnitOfWorkIntegration(t *testing.T) {
	pool := newBillingTestPool(t)
	ctx := context.Background()

	plans, err := postgres.NewPlanRepository(pool)
	if err != nil {
		t.Fatalf("create plan repository: %v", err)
	}
	subscriptions, err := postgres.NewSubscriptionRepository(pool)
	if err != nil {
		t.Fatalf("create subscription repository: %v", err)
	}
	outbox, err := postgres.NewSubscriptionOutboxRepository(pool)
	if err != nil {
		t.Fatalf("create outbox repository: %v", err)
	}
	unitOfWork, err := postgres.NewUnitOfWork(pool)
	if err != nil {
		t.Fatalf("create unit of work: %v", err)
	}

	t.Run("commits repository changes atomically", func(t *testing.T) {
		plan := newBillingTestPlan(t)
		subscription := newBillingTestSubscription(t, plan)

		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := plans.Create(txCtx, plan); err != nil {
				return err
			}
			if err := subscriptions.Create(txCtx, subscription); err != nil {
				return err
			}
			return outbox.Create(
				txCtx,
				"00000000-0000-4000-8000-000000000101",
				subscription.ID.String(),
				"billing.subscription.updated",
				[]byte(`{"status":"pending"}`),
			)
		})
		if err != nil {
			t.Fatalf("commit transaction: %v", err)
		}

		if _, err := plans.GetByID(ctx, plan.ID); err != nil {
			t.Fatalf("committed plan not found: %v", err)
		}
		if _, err := subscriptions.GetByID(ctx, subscription.ID); err != nil {
			t.Fatalf("committed subscription not found: %v", err)
		}

		pending, err := outbox.GetPending(ctx, 10)
		if err != nil {
			t.Fatalf("get committed outbox event: %v", err)
		}
		if len(pending) != 1 {
			t.Fatalf("committed outbox event count = %d, want 1", len(pending))
		}
	})

	t.Run("rolls back all repository changes when callback fails", func(t *testing.T) {
		plan := newBillingTestPlan(t)
		subscription := newBillingTestSubscription(t, plan)
		expectedErr := errors.New("deliberate transaction failure")

		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := plans.Create(txCtx, plan); err != nil {
				return err
			}
			if err := subscriptions.Create(txCtx, subscription); err != nil {
				return err
			}
			if err := outbox.Create(
				txCtx,
				"00000000-0000-4000-8000-000000000103",
				subscription.ID.String(),
				"billing.subscription.updated",
				[]byte(`{"status":"pending"}`),
			); err != nil {
				return err
			}
			return expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected callback error %v, got %v", expectedErr, err)
		}

		if _, err := plans.GetByID(ctx, plan.ID); !errors.Is(err, application.ErrPlanNotFound) {
			t.Fatalf("plan exists after rollback: %v", err)
		}
		if _, err := subscriptions.GetByID(ctx, subscription.ID); !errors.Is(err, application.ErrSubscriptionNotFound) {
			t.Fatalf("subscription exists after rollback: %v", err)
		}
	})

	t.Run("nested transaction reuses outer transaction", func(t *testing.T) {
		plan := newBillingTestPlan(t)
		expectedErr := errors.New("outer transaction failure")

		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := plans.Create(txCtx, plan); err != nil {
				return err
			}
			if err := unitOfWork.WithinTransaction(txCtx, func(nestedCtx context.Context) error {
				subscription := newBillingTestSubscription(t, plan)
				return subscriptions.Create(nestedCtx, subscription)
			}); err != nil {
				return err
			}
			return expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected outer transaction error %v, got %v", expectedErr, err)
		}

		if _, err := plans.GetByID(ctx, plan.ID); !errors.Is(err, application.ErrPlanNotFound) {
			t.Fatalf("nested transaction escaped outer rollback: %v", err)
		}
	})
}
