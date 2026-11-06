package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// SubscribeUseCase coordinates Store ownership validation and Billing
// checkout creation. Store plan and status are deliberately not modified here;
// they change only after the trusted subscription-success workflow is added.
type SubscribeUseCase struct {
	storeRepository  ports.StoreRepository
	identityProvider ports.IdentityProvider
	billingProvider  ports.BillingProvider
	logger           ports.Logger
	metrics          ports.Metrics
}

// NewSubscribeUseCase constructs the Store subscription checkout workflow.
func NewSubscribeUseCase(
	storeRepository ports.StoreRepository,
	identityProvider ports.IdentityProvider,
	billingProvider ports.BillingProvider,
	logger ports.Logger,
	metrics ports.Metrics,
) *SubscribeUseCase {
	return &SubscribeUseCase{
		storeRepository:  storeRepository,
		identityProvider: identityProvider,
		billingProvider:  billingProvider,
		logger:           logger,
		metrics:          metrics,
	}
}

// Execute validates the authenticated Store owner and requested Billing plan,
// then asks Billing to create or resume the subscription checkout.
func (uc *SubscribeUseCase) Execute(
	ctx context.Context,
	input dto.SubscribeInput,
) (dto.SubscribeOutput, error) {
	startedAt := time.Now()

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(ctx, startedAt, "authentication", applicationerrors.ErrUnauthenticated)
	}

	if input.StoreID == uuid.Nil || input.PlanID == uuid.Nil {
		return uc.fail(ctx, startedAt, "validation", applicationerrors.ErrInvalidSubscriptionRequest)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		if errors.Is(err, applicationerrors.ErrStoreNotFound) {
			return uc.fail(ctx, startedAt, "store_not_found", applicationerrors.ErrStoreNotFound)
		}
		return uc.fail(ctx, startedAt, "store_lookup", fmt.Errorf("find Store: %w", err))
	}
	if store == nil {
		return uc.fail(ctx, startedAt, "store_not_found", applicationerrors.ErrStoreNotFound)
	}

	// Do not reveal whether a Store exists when the caller does not own it.
	if !store.IsOwnedBy(identity.UserID) {
		return uc.fail(ctx, startedAt, "store_ownership", applicationerrors.ErrStoreNotFound)
	}

	billingPlan, err := uc.billingProvider.GetPlan(ctx, input.PlanID)
	if err != nil {
		return uc.fail(ctx, startedAt, "billing_plan_lookup", fmt.Errorf("get Billing plan: %w", err))
	}
	if billingPlan == nil || !billingPlan.Available {
		return uc.fail(ctx, startedAt, "billing_plan_unavailable", applicationerrors.ErrBillingPlanNotAvailable)
	}
	if billingPlan.ID != input.PlanID {
		return uc.fail(ctx, startedAt, "billing_plan_mismatch", applicationerrors.ErrInvalidSubscriptionRequest)
	}

	targetPlan, err := valueobjects.NewPlan(valueobjects.PlanType(strings.ToLower(strings.TrimSpace(billingPlan.Type))))
	if err != nil {
		return uc.fail(ctx, startedAt, "unsupported_plan", fmt.Errorf(
			"%w: %s",
			applicationerrors.ErrUnsupportedStorePlan,
			billingPlan.Type,
		))
	}

	if !targetPlan.CanAccommodateProducts(store.ProductCount()) {
		return uc.fail(ctx, startedAt, "product_capacity", applicationerrors.ErrStorePlanProductLimitExceeded)
	}

	owner, err := uc.identityProvider.GetUser(ctx, identity.UserID)
	if err != nil {
		return uc.fail(ctx, startedAt, "identity_lookup", fmt.Errorf("get Store owner: %w", err))
	}
	if owner == nil || owner.ID != identity.UserID || strings.TrimSpace(owner.Email) == "" {
		return uc.fail(ctx, startedAt, "identity_user_unavailable", applicationerrors.ErrIdentityUserUnavailable)
	}

	checkout, err := uc.billingProvider.Subscribe(ctx, ports.BillingSubscribeInput{
		StoreID:       store.ID().Value(),
		PayerID:       identity.UserID,
		PlanID:        input.PlanID,
		CustomerEmail: strings.TrimSpace(owner.Email),
	})
	if err != nil {
		return uc.fail(ctx, startedAt, "billing_subscribe", fmt.Errorf("create Billing subscription: %w", err))
	}
	if checkout == nil || strings.TrimSpace(checkout.CheckoutURL) == "" {
		return uc.fail(ctx, startedAt, "billing_checkout_unavailable", applicationerrors.ErrBillingCheckoutUnavailable)
	}

	uc.log(ctx, ports.LogEvent{
		Event:     "store.subscription.checkout_created",
		Operation: "subscribe",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})
	uc.incrementMetric(ctx, ports.Metric{
		Name:   "store_subscribe_success_total",
		Value:  1,
		Labels: map[string]string{"operation": "subscribe"},
	})
	uc.observeDuration(ctx, startedAt)

	return dto.SubscribeOutput{
		SubscriptionID: checkout.SubscriptionID,
		PaymentID:      checkout.PaymentID,
		CheckoutURL:    checkout.CheckoutURL,
	}, nil
}

func (uc *SubscribeUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	category string,
	err error,
) (dto.SubscribeOutput, error) {
	identity, _ := ports.AuthenticatedIdentityFromContext(ctx)

	uc.log(ctx, ports.LogEvent{
		Event:           "store.subscription.checkout_failed",
		Operation:       "subscribe",
		UserID:          identity.UserID.String(),
		Role:            identity.Role,
		FailureCategory: category,
	})
	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_subscribe_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "subscribe",
			"failure_category": category,
		},
	})
	uc.observeDuration(ctx, startedAt)

	return dto.SubscribeOutput{}, err
}

func (uc *SubscribeUseCase) log(ctx context.Context, event ports.LogEvent) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, event)
	}
}

func (uc *SubscribeUseCase) incrementMetric(ctx context.Context, metric ports.Metric) {
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, metric)
	}
}

func (uc *SubscribeUseCase) observeDuration(ctx context.Context, startedAt time.Time) {
	if uc.metrics != nil {
		_ = uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_subscribe_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
			Labels: map[string]string{
				"operation": "subscribe",
			},
		})
	}
}
