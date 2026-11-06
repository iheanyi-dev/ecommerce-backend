package kafka_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	kafka "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/messaging/kafka"
	"go.uber.org/fx/fxtest"
)

type fakeOutboxRepository struct {
	events    []ports.SubscriptionOutboxEvent
	getErr    error
	markErr   error
	recordErr error
	published []string
	failed    []string
}

func (f *fakeOutboxRepository) Create(context.Context, string, string, string, []byte) error {
	return nil
}

func (f *fakeOutboxRepository) GetPending(context.Context, int32) ([]ports.SubscriptionOutboxEvent, error) {
	return f.events, f.getErr
}

func (f *fakeOutboxRepository) MarkPublished(_ context.Context, id string) error {
	f.published = append(f.published, id)
	return f.markErr
}

func (f *fakeOutboxRepository) RecordFailure(_ context.Context, id string, _ error) error {
	f.failed = append(f.failed, id)
	return f.recordErr
}

type fakeEventPublisher struct {
	err       error
	published []ports.SubscriptionOutboxEvent
}

func (f *fakeEventPublisher) Publish(_ context.Context, event ports.SubscriptionOutboxEvent) error {
	f.published = append(f.published, event)
	return f.err
}

func TestOutboxWorkerProcessPending(t *testing.T) {
	ctx := context.Background()

	t.Run("publishes and marks events", func(t *testing.T) {
		repository := &fakeOutboxRepository{
			events: []ports.SubscriptionOutboxEvent{
				{ID: "event-1", Payload: []byte(`{"status":"active"}`)},
			},
		}
		publisher := &fakeEventPublisher{}
		worker := kafka.NewOutboxWorker(fxtest.NewLifecycle(t), repository, publisher)

		worker.ProcessPending(ctx)

		if len(publisher.published) != 1 || publisher.published[0].ID != "event-1" {
			t.Fatalf("published events = %+v", publisher.published)
		}
		if len(repository.published) != 1 || repository.published[0] != "event-1" {
			t.Fatalf("marked published = %v", repository.published)
		}
		if len(repository.failed) != 0 {
			t.Fatalf("unexpected failures = %v", repository.failed)
		}
	})

	t.Run("records publish failure", func(t *testing.T) {
		repository := &fakeOutboxRepository{
			events: []ports.SubscriptionOutboxEvent{
				{ID: "event-2", Payload: []byte(`{"status":"active"}`)},
			},
		}
		publisher := &fakeEventPublisher{err: errors.New("broker unavailable")}
		worker := kafka.NewOutboxWorker(fxtest.NewLifecycle(t), repository, publisher)

		worker.ProcessPending(ctx)

		if len(repository.failed) != 1 || repository.failed[0] != "event-2" {
			t.Fatalf("recorded failures = %v", repository.failed)
		}
		if len(repository.published) != 0 {
			t.Fatalf("unexpected published events = %v", repository.published)
		}
	})

	t.Run("stops when pending events cannot be loaded", func(t *testing.T) {
		repository := &fakeOutboxRepository{getErr: errors.New("database unavailable")}
		publisher := &fakeEventPublisher{}
		worker := kafka.NewOutboxWorker(fxtest.NewLifecycle(t), repository, publisher)

		worker.ProcessPending(ctx)

		if len(publisher.published) != 0 {
			t.Fatalf("unexpected published events = %+v", publisher.published)
		}
	})

	t.Run("continues when marking an event fails", func(t *testing.T) {
		repository := &fakeOutboxRepository{
			events: []ports.SubscriptionOutboxEvent{
				{ID: "event-3", Payload: []byte(`{"status":"active"}`)},
			},
			markErr: errors.New("database unavailable"),
		}
		publisher := &fakeEventPublisher{}
		worker := kafka.NewOutboxWorker(fxtest.NewLifecycle(t), repository, publisher)

		worker.ProcessPending(ctx)

		if len(publisher.published) != 1 {
			t.Fatalf("published events = %+v", publisher.published)
		}
		if len(repository.failed) != 0 {
			t.Fatalf("unexpected recorded failures = %v", repository.failed)
		}
	})
}
