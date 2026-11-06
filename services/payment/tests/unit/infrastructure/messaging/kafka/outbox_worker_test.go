package kafka_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	kafka "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/messaging/kafka"
	"go.uber.org/fx"
)

type fakeLifecycle struct{}

func (fakeLifecycle) Append(fx.Hook) {}

type fakeOutboxRepository struct {
	events       []ports.PaymentOutboxEvent
	getErr       error
	markErr      error
	failureErr   error
	marked       []string
	failed       []string
	failureCause error
	getCalls     int
	cancel       context.CancelFunc
}

func (r *fakeOutboxRepository) Create(context.Context, string, string, string, string, string, []byte) error {
	return nil
}

func (r *fakeOutboxRepository) GetPending(_ context.Context, limit int32) ([]ports.PaymentOutboxEvent, error) {
	r.getCalls++
	if r.getErr != nil {
		if r.cancel != nil {
			r.cancel()
		}
		return nil, r.getErr
	}
	if int32(len(r.events)) > limit {
		return r.events[:limit], nil
	}
	return r.events, nil
}

func (r *fakeOutboxRepository) MarkPublished(_ context.Context, id string) error {
	r.marked = append(r.marked, id)
	return r.markErr
}

func (r *fakeOutboxRepository) RecordFailure(_ context.Context, id string, cause error) error {
	r.failed = append(r.failed, id)
	r.failureCause = cause
	return r.failureErr
}

type fakeEventPublisher struct {
	errors      map[string]error
	calls       []string
	cancel      context.CancelFunc
	cancelAfter int
}

func (p *fakeEventPublisher) Publish(_ context.Context, event ports.PaymentOutboxEvent) error {
	p.calls = append(p.calls, event.ID)
	if p.cancel != nil && len(p.calls) == p.cancelAfter {
		p.cancel()
	}
	return p.errors[event.ID]
}

func TestOutboxWorkerPublishesAndMarksEvents(t *testing.T) {
	repository := &fakeOutboxRepository{
		events: []ports.PaymentOutboxEvent{
			{ID: "event-1", Payload: []byte(`{"status":"succeeded"}`)},
			{ID: "event-2", Payload: []byte(`{"status":"failed"}`)},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	publisher := &fakeEventPublisher{
		errors:      map[string]error{},
		cancel:      cancel,
		cancelAfter: 2,
	}
	worker := kafka.NewOutboxWorker(fakeLifecycle{}, repository, publisher)

	worker.Run(ctx)

	if len(publisher.calls) != 2 {
		t.Fatalf("publish calls = %d, want 2", len(publisher.calls))
	}
	if len(repository.marked) != 2 {
		t.Fatalf("marked events = %v, want both events", repository.marked)
	}
	if len(repository.failed) != 0 {
		t.Errorf("failed events = %v, want none", repository.failed)
	}
}

func TestOutboxWorkerRecordsPublishFailureAndContinues(t *testing.T) {
	publishErr := errors.New("Kafka unavailable")
	repository := &fakeOutboxRepository{
		events: []ports.PaymentOutboxEvent{
			{ID: "event-1", Payload: []byte(`{"status":"succeeded"}`)},
			{ID: "event-2", Payload: []byte(`{"status":"failed"}`)},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	publisher := &fakeEventPublisher{
		errors:      map[string]error{"event-1": publishErr},
		cancel:      cancel,
		cancelAfter: 2,
	}
	worker := kafka.NewOutboxWorker(fakeLifecycle{}, repository, publisher)

	worker.Run(ctx)

	if len(publisher.calls) != 2 {
		t.Fatalf("publish calls = %d, want 2", len(publisher.calls))
	}
	if len(repository.failed) != 1 || repository.failed[0] != "event-1" {
		t.Fatalf("failed events = %v, want [event-1]", repository.failed)
	}
	if !errors.Is(repository.failureCause, publishErr) {
		t.Errorf("recorded failure = %v, want %v", repository.failureCause, publishErr)
	}
	if len(repository.marked) != 1 || repository.marked[0] != "event-2" {
		t.Errorf("marked events = %v, want [event-2]", repository.marked)
	}
}

func TestOutboxWorkerStopsWhenPendingEventsCannotBeLoaded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &fakeOutboxRepository{
		getErr: errors.New("database unavailable"),
		cancel: cancel,
	}
	publisher := &fakeEventPublisher{errors: map[string]error{}}
	worker := kafka.NewOutboxWorker(fakeLifecycle{}, repository, publisher)

	worker.Run(ctx)

	if repository.getCalls != 1 {
		t.Errorf("GetPending calls = %d, want 1", repository.getCalls)
	}
	if len(publisher.calls) != 0 {
		t.Errorf("publish calls = %d, want 0", len(publisher.calls))
	}
}

func TestOutboxWorkerSkipsEventsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := &fakeOutboxRepository{
		events: []ports.PaymentOutboxEvent{
			{ID: "event-1", Payload: []byte(`{"status":"succeeded"}`)},
		},
	}
	publisher := &fakeEventPublisher{errors: map[string]error{}}
	worker := kafka.NewOutboxWorker(fakeLifecycle{}, repository, publisher)

	worker.Run(ctx)

	if len(publisher.calls) != 0 {
		t.Errorf("publish calls = %d, want 0", len(publisher.calls))
	}
}
