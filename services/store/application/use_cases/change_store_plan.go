package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// ErrStorePlanProductLimitExceeded indicates that the Store currently contains
// more products than the requested plan permits.
//
// Product capacity is an application-level transition rule because the
// application obtains the current product count through the ProductCountProvider
// port before allowing the Store aggregate to change plan.
var ErrStorePlanProductLimitExceeded = fmt.Errorf(
	"store plan product limit exceeded",
)

// ChangeStorePlanUseCase coordinates an authenticated Store owner's plan
// transition.
//
// The workflow deliberately keeps product ownership outside the Store
// aggregate. Store owns the plan's capabilities, while the application layer
// obtains the current product count through ProductCountProvider before
// permitting a transition.
//
// A plan change is persisted through StoreRepository.ChangePlan because the
// transition may change more than the plan itself. In the current contract,
// moving to Premium also deactivates the Store.
type ChangeStorePlanUseCase struct {
	storeRepository      ports.StoreRepository
	productCountProvider ports.ProductCountProvider
	logger               ports.Logger
	metrics              ports.Metrics
}

// NewChangeStorePlanUseCase constructs the Change Store Plan use case.
func NewChangeStorePlanUseCase(
	storeRepository ports.StoreRepository,
	productCountProvider ports.ProductCountProvider,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChangeStorePlanUseCase {
	return &ChangeStorePlanUseCase{
		storeRepository:      storeRepository,
		productCountProvider: productCountProvider,
		logger:               logger,
		metrics:              metrics,
	}
}

// Execute changes the plan of the Store owned by the authenticated caller.
func (uc *ChangeStorePlanUseCase) Execute(
	ctx context.Context,
	input dto.ChangeStorePlanInput,
) (dto.StoreOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_plan_total",
		Value: 1,
	})

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			"unauthenticated",
			applicationerrors.ErrUnauthenticated,
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_lookup",
			fmt.Errorf("find store: %w", err),
		)
	}

	if store == nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	if !store.IsOwnedBy(identity.UserID) {
		// Do not reveal whether a Store exists to a different authenticated
		// user. Ownership failure is represented as not found.
		return uc.fail(
			ctx,
			startedAt,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	targetPlan, err := valueobjects.NewPlan(
		valueobjects.PlanType(input.PlanType),
	)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"invalid_plan",
			err,
		)
	}

	productCount, err := uc.productCountProvider.CountByStoreID(
		ctx,
		store.ID().Value(),
	)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"product_count",
			fmt.Errorf("count store products: %w", err),
		)
	}

	if !targetPlan.CanAccommodateProducts(productCount) {
		return uc.fail(
			ctx,
			startedAt,
			"plan_product_limit",
			ErrStorePlanProductLimitExceeded,
		)
	}

	// Change the plan through domain behavior so the aggregate remains
	// responsible for its own state and timestamp invariants.
	if err := store.ChangePlan(targetPlan); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"plan_change",
			err,
		)
	}

	// Premium plan transitions deliberately deactivate the Store. Subscription
	// or Billing is responsible for the later reactivation event.
	if targetPlan.Type() != valueobjects.PlanTypeBasic {
		if err := store.Deactivate(); err != nil {
			return uc.fail(
				ctx,
				startedAt,
				"status_transition",
				err,
			)
		}
	}

	// ChangePlan persists the complete aggregate because the transition may
	// contain both plan and status changes.
	if err := uc.storeRepository.ChangePlan(ctx, store); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_persistence",
			fmt.Errorf("change store plan: %w", err),
		)
	}

	output := toStoreOutput(store)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.change_plan.succeeded",
		Operation: "change_store_plan",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_plan_success_total",
		Value: 1,
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_change_plan_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return output, nil
}

func (uc *ChangeStorePlanUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.StoreOutput, error) {
	identity, _ := ports.AuthenticatedIdentityFromContext(ctx)

	uc.log(ctx, ports.LogEvent{
		Event:           "store.change_plan.failed",
		Operation:       "change_store_plan",
		UserID:          identity.UserID.String(),
		Role:            identity.Role,
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_plan_failure_total",
		Value: 1,
		Labels: map[string]string{
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_change_plan_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return dto.StoreOutput{}, err
}

func (uc *ChangeStorePlanUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

func (uc *ChangeStorePlanUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *ChangeStorePlanUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, metric)
}
