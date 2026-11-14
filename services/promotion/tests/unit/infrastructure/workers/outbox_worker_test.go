package workers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/workers"
	"go.uber.org/fx"
)

type outboxRepositoryStub struct {
	events     []ports.PendingPromotionEvent
	claimErr   error
	published  []string
	failed     []string
	publishErr error
	markErr    error
	recordErr  error
}

func (r *outboxRepositoryStub) Create(
	context.Context, string, string, string, []byte,
) error {
	return nil
}

func (r *outboxRepositoryStub) ClaimPending(
	context.Context, int, time.Duration,
) ([]ports.PendingPromotionEvent, error) {
	return r.events, r.claimErr
}

func (r *outboxRepositoryStub) MarkPublished(
	_ context.Context, eventID, token string,
) error {
	r.published = append(r.published, eventID+":"+token)
	return r.markErr
}

func (r *outboxRepositoryStub) RecordFailure(
	_ context.Context, eventID, token string, _ error,
) error {
	r.failed = append(r.failed, eventID+":"+token)
	return r.recordErr
}

type publisherStub struct {
	calls   []string
	failFor map[string]error
}

func (p *publisherStub) Publish(
	_ context.Context,
	event ports.PendingPromotionEvent,
) error {
	p.calls = append(p.calls, event.EventID)
	return p.failFor[event.EventID]
}

func TestOutboxWorkerProcessPending(t *testing.T) {
	tests := []struct {
		name          string
		events        []ports.PendingPromotionEvent
		failFor       map[string]error
		claimErr      error
		markErr       error
		recordErr     error
		wantPublished []string
		wantFailed    []string
		wantCalls     []string
		wantErr       bool
	}{
		{
			name: "publishes and acknowledges events",
			events: []ports.PendingPromotionEvent{
				{EventID: "event-1", ClaimToken: "claim-1"},
				{EventID: "event-2", ClaimToken: "claim-2"},
			},
			wantCalls:     []string{"event-1", "event-2"},
			wantPublished: []string{"event-1:claim-1", "event-2:claim-2"},
		},
		{
			name: "records publish failure and continues batch",
			events: []ports.PendingPromotionEvent{
				{EventID: "event-1", ClaimToken: "claim-1"},
				{EventID: "event-2", ClaimToken: "claim-2"},
			},
			failFor: map[string]error{
				"event-1": errors.New("broker unavailable"),
			},
			wantCalls:     []string{"event-1", "event-2"},
			wantFailed:    []string{"event-1:claim-1"},
			wantPublished: []string{"event-2:claim-2"},
		},
		{
			name:     "returns claim error",
			claimErr: errors.New("database unavailable"),
			wantErr:  true,
		},
		{
			name: "returns acknowledgement error",
			events: []ports.PendingPromotionEvent{
				{EventID: "event-1", ClaimToken: "claim-1"},
			},
			markErr:       errors.New("database unavailable"),
			wantCalls:     []string{"event-1"},
			wantPublished: []string{"event-1:claim-1"},
			wantErr:       true,
		},
		{
			name: "returns failure recording error",
			events: []ports.PendingPromotionEvent{
				{EventID: "event-1", ClaimToken: "claim-1"},
			},
			failFor: map[string]error{
				"event-1": errors.New("broker unavailable"),
			},
			recordErr:  errors.New("database unavailable"),
			wantCalls:  []string{"event-1"},
			wantFailed: []string{"event-1:claim-1"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &outboxRepositoryStub{
				events:    tt.events,
				claimErr:  tt.claimErr,
				markErr:   tt.markErr,
				recordErr: tt.recordErr,
			}
			publisher := &publisherStub{failFor: tt.failFor}

			worker := newWorker(t, repository, publisher)
			err := worker.ProcessPending(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			assertStrings(t, "publish calls", publisher.calls, tt.wantCalls)
			assertStrings(t, "published", repository.published, tt.wantPublished)
			assertStrings(t, "failed", repository.failed, tt.wantFailed)
		})
	}
}

type lifecycleStub struct{}

func (*lifecycleStub) Append(fx.Hook) {}

func newWorker(
	t *testing.T,
	repository *outboxRepositoryStub,
	publisher *publisherStub,
) *workers.OutboxWorker {
	t.Helper()

	worker, err := workers.NewOutboxWorker(&lifecycleStub{}, repository, publisher)
	if err != nil {
		t.Fatalf("create outbox worker: %v", err)
	}
	return worker
}

func assertStrings(t *testing.T, name string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
	}
}
