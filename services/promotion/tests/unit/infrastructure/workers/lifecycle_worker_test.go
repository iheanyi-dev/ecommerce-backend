package workers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/workers"
)

type fakePromotionRepository struct {
	scheduledIDs   []string
	expiredIDs     []string
	scheduledErr   error
	expiredErr     error
	scheduledNow   time.Time
	expiredNow     time.Time
	scheduledLimit int
	expiredLimit   int
}

func (f *fakePromotionRepository) ListScheduledDue(
	_ context.Context, now time.Time, limit int,
) ([]string, error) {
	f.scheduledNow, f.scheduledLimit = now, limit
	return f.scheduledIDs, f.scheduledErr
}

func (f *fakePromotionRepository) ListExpiredDue(
	_ context.Context, now time.Time, limit int,
) ([]string, error) {
	f.expiredNow, f.expiredLimit = now, limit
	return f.expiredIDs, f.expiredErr
}

type fakeActivator struct {
	ids  []string
	fail map[string]error
}

func (f *fakeActivator) Execute(_ context.Context, input dtos.ActivatePromotionInput) (dtos.PromotionDTO, error) {
	f.ids = append(f.ids, input.PromotionID)
	return dtos.PromotionDTO{}, f.fail[input.PromotionID]
}

type fakeExpirer struct {
	ids  []string
	fail map[string]error
}

func (f *fakeExpirer) Execute(_ context.Context, input dtos.ExpirePromotionInput) (dtos.PromotionDTO, error) {
	f.ids = append(f.ids, input.PromotionID)
	return dtos.PromotionDTO{}, f.fail[input.PromotionID]
}

func TestLifecycleWorkerProcessDue(t *testing.T) {
	repository := &fakePromotionRepository{
		scheduledIDs: []string{"scheduled-1", "scheduled-2"},
		expiredIDs:   []string{"expired-1"},
	}
	activator := &fakeActivator{}
	expirer := &fakeExpirer{}

	worker := workers.NewLifecycleWorkerRunner(repository, activator, expirer)
	worker.ProcessDue(context.Background())

	if len(activator.ids) != 2 || activator.ids[0] != "scheduled-1" {
		t.Fatalf("unexpected activation calls: %v", activator.ids)
	}
	if len(expirer.ids) != 1 || expirer.ids[0] != "expired-1" {
		t.Fatalf("unexpected expiration calls: %v", expirer.ids)
	}
	if repository.scheduledLimit != 100 ||
		repository.expiredLimit != 100 {
		t.Fatal("worker did not use the configured batch size")
	}
	if repository.scheduledNow.IsZero() || repository.expiredNow.IsZero() {
		t.Fatal("worker did not supply query timestamps")
	}
}

func TestLifecycleWorkerContinuesAfterTransitionFailure(t *testing.T) {
	repository := &fakePromotionRepository{
		scheduledIDs: []string{"bad-activation", "good-activation"},
		expiredIDs:   []string{"bad-expiration", "good-expiration"},
	}
	activator := &fakeActivator{fail: map[string]error{
		"bad-activation": errors.New("activation failed"),
	}}
	expirer := &fakeExpirer{fail: map[string]error{
		"bad-expiration": errors.New("expiration failed"),
	}}

	worker := workers.NewLifecycleWorkerRunner(repository, activator, expirer)
	worker.ProcessDue(context.Background())

	if len(activator.ids) != 2 {
		t.Fatalf("worker stopped after activation failure: %v", activator.ids)
	}
	if len(expirer.ids) != 2 {
		t.Fatalf("worker stopped after expiration failure: %v", expirer.ids)
	}
}

func TestLifecycleWorkerSkipsWorkWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := &fakePromotionRepository{
		scheduledIDs: []string{"scheduled-1"},
		expiredIDs:   []string{"expired-1"},
	}
	activator := &fakeActivator{}
	expirer := &fakeExpirer{}

	worker := workers.NewLifecycleWorkerRunner(repository, activator, expirer)
	worker.ProcessDue(ctx)

	if len(activator.ids) != 0 || len(expirer.ids) != 0 {
		t.Fatal("worker processed promotions with a cancelled context")
	}
}
