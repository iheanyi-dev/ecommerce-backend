package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// ListBillingPlansUseCase retrieves the Billing plan catalogue.
type ListBillingPlansUseCase struct {
	billingProvider ports.BillingProvider
}

// NewListBillingPlansUseCase constructs the plan catalogue query.
func NewListBillingPlansUseCase(provider ports.BillingProvider) *ListBillingPlansUseCase {
	return &ListBillingPlansUseCase{billingProvider: provider}
}

// Execute returns all plans configured in Billing.
func (uc *ListBillingPlansUseCase) Execute(
	ctx context.Context,
) ([]dto.BillingPlanOutput, error) {
	plans, err := uc.billingProvider.ListPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("list Billing plans: %w", err)
	}

	output := make([]dto.BillingPlanOutput, 0, len(plans))
	for _, plan := range plans {
		output = append(output, toBillingPlanOutput(plan))
	}
	return output, nil
}

// GetBillingPlanUseCase retrieves one Billing plan.
type GetBillingPlanUseCase struct {
	billingProvider ports.BillingProvider
}

// NewGetBillingPlanUseCase constructs the plan lookup query.
func NewGetBillingPlanUseCase(provider ports.BillingProvider) *GetBillingPlanUseCase {
	return &GetBillingPlanUseCase{billingProvider: provider}
}

// Execute retrieves a plan by its identifier.
func (uc *GetBillingPlanUseCase) Execute(
	ctx context.Context,
	input dto.GetBillingPlanInput,
) (dto.BillingPlanOutput, error) {
	if input.PlanID == uuid.Nil {
		return dto.BillingPlanOutput{}, applicationerrors.ErrInvalidSubscriptionRequest
	}

	plan, err := uc.billingProvider.GetPlan(ctx, input.PlanID)
	if err != nil {
		return dto.BillingPlanOutput{}, fmt.Errorf("get Billing plan: %w", err)
	}
	if plan == nil || plan.ID != input.PlanID {
		return dto.BillingPlanOutput{}, applicationerrors.ErrBillingPlanNotAvailable
	}

	return toBillingPlanOutput(*plan), nil
}

// ListStoreSubscriptionsUseCase retrieves the subscription history of an
// authenticated Store owner.
type ListStoreSubscriptionsUseCase struct {
	storeRepository ports.StoreRepository
	billingProvider ports.BillingProvider
}

// NewListStoreSubscriptionsUseCase constructs the Store subscription query.
func NewListStoreSubscriptionsUseCase(
	repository ports.StoreRepository,
	provider ports.BillingProvider,
) *ListStoreSubscriptionsUseCase {
	return &ListStoreSubscriptionsUseCase{
		storeRepository: repository,
		billingProvider: provider,
	}
}

// Execute verifies Store ownership before retrieving subscription history.
func (uc *ListStoreSubscriptionsUseCase) Execute(
	ctx context.Context,
	input dto.ListStoreSubscriptionsInput,
) ([]dto.BillingSubscriptionOutput, error) {
	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return nil, applicationerrors.ErrUnauthenticated
	}
	if input.StoreID == uuid.Nil {
		return nil, applicationerrors.ErrInvalidSubscriptionRequest
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		if errors.Is(err, applicationerrors.ErrStoreNotFound) {
			return nil, applicationerrors.ErrStoreNotFound
		}
		return nil, fmt.Errorf("find Store for subscription history: %w", err)
	}
	if store == nil || !store.IsOwnedBy(identity.UserID) {
		return nil, applicationerrors.ErrStoreNotFound
	}

	subscriptions, err := uc.billingProvider.ListStoreSubscriptions(ctx, input.StoreID)
	if err != nil {
		return nil, fmt.Errorf("list Store subscriptions: %w", err)
	}

	output := make([]dto.BillingSubscriptionOutput, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		// Do not return data that Billing associates with a different Store.
		if subscription.StoreID != input.StoreID {
			return nil, applicationerrors.ErrBillingSubscriptionNotFound
		}
		output = append(output, toBillingSubscriptionOutput(subscription))
	}
	return output, nil
}

// GetStoreSubscriptionUseCase retrieves one subscription after verifying
// that the caller owns the requested Store.
type GetStoreSubscriptionUseCase struct {
	storeRepository ports.StoreRepository
	billingProvider ports.BillingProvider
}

// NewGetStoreSubscriptionUseCase constructs the subscription lookup query.
func NewGetStoreSubscriptionUseCase(
	repository ports.StoreRepository,
	provider ports.BillingProvider,
) *GetStoreSubscriptionUseCase {
	return &GetStoreSubscriptionUseCase{
		storeRepository: repository,
		billingProvider: provider,
	}
}

// Execute retrieves a subscription and verifies that it belongs to the
// requested Store.
func (uc *GetStoreSubscriptionUseCase) Execute(
	ctx context.Context,
	input dto.GetStoreSubscriptionInput,
) (dto.BillingSubscriptionOutput, error) {
	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return dto.BillingSubscriptionOutput{}, applicationerrors.ErrUnauthenticated
	}
	if input.StoreID == uuid.Nil || input.SubscriptionID == uuid.Nil {
		return dto.BillingSubscriptionOutput{}, applicationerrors.ErrInvalidSubscriptionRequest
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		if errors.Is(err, applicationerrors.ErrStoreNotFound) {
			return dto.BillingSubscriptionOutput{}, applicationerrors.ErrStoreNotFound
		}
		return dto.BillingSubscriptionOutput{}, fmt.Errorf("find Store for subscription: %w", err)
	}
	if store == nil || !store.IsOwnedBy(identity.UserID) {
		return dto.BillingSubscriptionOutput{}, applicationerrors.ErrStoreNotFound
	}

	subscription, err := uc.billingProvider.GetSubscription(ctx, input.SubscriptionID)
	if err != nil {
		return dto.BillingSubscriptionOutput{}, fmt.Errorf("get Store subscription: %w", err)
	}
	if subscription == nil || subscription.ID != input.SubscriptionID ||
		subscription.StoreID != input.StoreID {
		return dto.BillingSubscriptionOutput{}, applicationerrors.ErrBillingSubscriptionNotFound
	}

	return toBillingSubscriptionOutput(*subscription), nil
}

func toBillingPlanOutput(plan ports.BillingPlan) dto.BillingPlanOutput {
	output := dto.BillingPlanOutput{
		ID:          plan.ID,
		Type:        plan.Type,
		Name:        plan.Name,
		Description: plan.Description,
		Price: dto.BillingMoneyOutput{
			AmountMinor: plan.Price.AmountMinor,
			Currency:    plan.Price.Currency,
		},
		Duration:    plan.Duration,
		MaxProducts: plan.Capabilities.MaxProducts,
		AIEnabled:   plan.Capabilities.AIEnabled,
		Available:   plan.Available,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}
	if plan.Promotion != nil {
		output.Promotion = &dto.BillingPromotionOutput{
			Name:                plan.Promotion.Name,
			DiscountAmountMinor: plan.Promotion.DiscountAmountMinor,
			StartsAt:            plan.Promotion.StartsAt,
			EndsAt:              plan.Promotion.EndsAt,
		}
	}
	return output
}

func toBillingSubscriptionOutput(
	subscription ports.BillingSubscription,
) dto.BillingSubscriptionOutput {
	return dto.BillingSubscriptionOutput{
		ID:      subscription.ID,
		StoreID: subscription.StoreID,
		PlanID:  subscription.PlanID,
		Price: dto.BillingMoneyOutput{
			AmountMinor: subscription.Price.AmountMinor,
			Currency:    subscription.Price.Currency,
		},
		Duration:  subscription.Duration,
		Status:    subscription.Status,
		StartedAt: subscription.StartedAt,
		ExpiresAt: subscription.ExpiresAt,
		CreatedAt: subscription.CreatedAt,
		UpdatedAt: subscription.UpdatedAt,
	}
}
