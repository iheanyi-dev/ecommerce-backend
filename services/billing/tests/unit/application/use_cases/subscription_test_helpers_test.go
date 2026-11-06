package use_cases_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

type fakeSubscriptionRepository struct {
	subscriptions []*entities.Subscription
	current       *entities.Subscription
	getErr        error
	createErr     error
	updateErr     error
	created       *entities.Subscription
	updated       *entities.Subscription
	getCalls      int
}

var _ ports.SubscriptionRepository = (*fakeSubscriptionRepository)(nil)

func (repository *fakeSubscriptionRepository) GetCurrentByStore(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Subscription, error) {
	repository.getCalls++
	if repository.getErr != nil {
		return nil, repository.getErr
	}
	if repository.current == nil {
		return nil, application.ErrSubscriptionNotFound
	}

	copy := *repository.current
	return &copy, nil
}

func (repository *fakeSubscriptionRepository) GetDueForUpdate(
	_ context.Context,
	now time.Time,
	limit int32,
) ([]entities.Subscription, error) {
	if repository.getErr != nil {
		return nil, repository.getErr
	}

	subscriptions := repository.subscriptions
	if subscriptions == nil && repository.current != nil {
		subscriptions = []*entities.Subscription{repository.current}
	}

	result := make([]entities.Subscription, 0)
	for _, subscription := range subscriptions {
		if subscription == nil ||
			subscription.Status != entities.SubscriptionStatusActive ||
			subscription.ExpiresAt.After(now) {
			continue
		}
		result = append(result, *subscription)
		if int32(len(result)) >= limit {
			break
		}
	}
	return result, nil
}

func (repository *fakeSubscriptionRepository) Create(
	_ context.Context,
	subscription entities.Subscription,
) error {
	if repository.createErr != nil {
		return repository.createErr
	}

	copy := subscription
	repository.created = &copy
	repository.current = &copy
	return nil
}

func (repository *fakeSubscriptionRepository) Update(
	_ context.Context,
	subscription entities.Subscription,
) error {
	if repository.updateErr != nil {
		return repository.updateErr
	}

	copy := subscription
	repository.updated = &copy
	repository.current = &copy
	return nil
}

type fakeSubscriptionPayment struct {
	result  ports.InitiatePaymentResult
	err     error
	request ports.InitiatePaymentRequest
	calls   int
}

var _ ports.Payment = (*fakeSubscriptionPayment)(nil)

func (payment *fakeSubscriptionPayment) InitiatePayment(
	_ context.Context,
	request ports.InitiatePaymentRequest,
) (ports.InitiatePaymentResult, error) {
	payment.calls++
	payment.request = request
	return payment.result, payment.err
}

// GetByID returns the fake repository's current subscription when its ID matches.
func (repository *fakeSubscriptionRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*entities.Subscription, error) {
	if repository.getErr != nil {
		return nil, repository.getErr
	}
	if repository.current == nil || repository.current.ID != id {
		return nil, application.ErrSubscriptionNotFound
	}
	subscription := *repository.current
	return &subscription, nil
}

// GetByStoreID returns all fake subscriptions belonging to the requested store.
func (repository *fakeSubscriptionRepository) GetByStoreID(
	ctx context.Context,
	storeID uuid.UUID,
) ([]entities.Subscription, error) {
	if repository.getErr != nil {
		return nil, repository.getErr
	}

	subscriptions := repository.subscriptions
	if subscriptions == nil && repository.current != nil {
		subscriptions = []*entities.Subscription{repository.current}
	}

	result := make([]entities.Subscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription != nil && subscription.StoreID == storeID {
			result = append(result, *subscription)
		}
	}
	return result, nil
}

type fakeUnitOfWork struct {
	calls int
	err   error
}

var _ ports.UnitOfWork = (*fakeUnitOfWork)(nil)

func (uow *fakeUnitOfWork) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	uow.calls++
	if uow.err != nil {
		return uow.err
	}
	return fn(ctx)
}

type fakeSubscriptionOutboxRepository struct {
	eventID        string
	subscriptionID string
	eventType      string
	payload        []byte
	calls          int
	err            error
}

func (fake *fakeSubscriptionOutboxRepository) GetPending(
	context.Context,
	int32,
) ([]ports.SubscriptionOutboxEvent, error) {
	return nil, nil
}

func (fake *fakeSubscriptionOutboxRepository) MarkPublished(context.Context, string) error {
	return nil
}

func (fake *fakeSubscriptionOutboxRepository) RecordFailure(context.Context, string, error) error {
	return nil
}

var _ ports.SubscriptionOutboxRepository = (*fakeSubscriptionOutboxRepository)(nil)

func (repository *fakeSubscriptionOutboxRepository) Create(
	_ context.Context,
	eventID string,
	subscriptionID string,
	eventType string,
	payload []byte,
) error {
	repository.calls++
	repository.eventID = eventID
	repository.subscriptionID = subscriptionID
	repository.eventType = eventType
	repository.payload = payload
	return repository.err
}
