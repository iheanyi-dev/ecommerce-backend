package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// RemoveStaleKnowledgeUseCase requests stale knowledge cleanup from the AI
// service.
//
// Store authorizes the request and verifies that the Store has AI features.
// The AI service owns knowledge storage and decides which knowledge is stale.
type RemoveStaleKnowledgeUseCase struct {
	storeRepository ports.StoreRepository
	dispatcher      ports.StaleKnowledgeDispatcher
	logger          ports.Logger
	metrics         ports.Metrics
}

func NewRemoveStaleKnowledgeUseCase(
	storeRepository ports.StoreRepository,
	dispatcher ports.StaleKnowledgeDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *RemoveStaleKnowledgeUseCase {
	return &RemoveStaleKnowledgeUseCase{
		storeRepository: storeRepository,
		dispatcher:      dispatcher,
		logger:          logger,
		metrics:         metrics,
	}
}

func (uc *RemoveStaleKnowledgeUseCase) Execute(
	ctx context.Context,
	input dto.RemoveStaleKnowledgeInput,
) (dto.RemoveStaleKnowledgeOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:   "store_remove_stale_knowledge_total",
		Value:  1,
		Labels: map[string]string{"operation": "remove_stale_knowledge"},
	})

	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)
	if !authenticated {
		return uc.fail(
			ctx, startedAt, "authentication",
			applicationerrors.ErrUnauthenticated,
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx, startedAt, "repository_lookup",
			fmt.Errorf("find store: %w", err),
		)
	}

	if store == nil || !store.IsOwnedBy(identity.UserID) {
		return uc.fail(
			ctx, startedAt, "authorization",
			applicationerrors.ErrStoreNotFound,
		)
	}

	if !store.Plan().AIFeaturesAllowed() {
		return uc.fail(
			ctx, startedAt, "ai_not_allowed",
			fmt.Errorf("store plan does not allow AI features"),
		)
	}

	err = uc.dispatcher.DispatchRemoveStaleKnowledge(
		ctx,
		ports.StaleKnowledge{
			StoreID:  store.ID().Value(),
			Filename: input.Filename,
		},
	)
	if err != nil {
		return uc.fail(
			ctx, startedAt, "knowledge_dispatch",
			fmt.Errorf("dispatch remove stale knowledge: %w", err),
		)
	}

	_ = uc.log(ctx, ports.LogEvent{
		Event:     "store.remove_stale_knowledge.succeeded",
		Operation: "remove_stale_knowledge",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.increment(ctx, ports.Metric{
		Name:   "store_remove_stale_knowledge_success_total",
		Value:  1,
		Labels: map[string]string{"operation": "remove_stale_knowledge"},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_remove_stale_knowledge_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "remove_stale_knowledge",
			"result":    "success",
		},
	})

	return dto.RemoveStaleKnowledgeOutput{
		StoreID:  store.ID().Value(),
		Filename: input.Filename,
	}, nil
}

func (uc *RemoveStaleKnowledgeUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.RemoveStaleKnowledgeOutput, error) {
	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)

	userID := ""
	role := ""

	if authenticated {
		userID = identity.UserID.String()
		role = identity.Role
	}

	_ = uc.log(ctx, ports.LogEvent{
		Event:           "store.remove_stale_knowledge.failed",
		Operation:       "remove_stale_knowledge",
		UserID:          userID,
		Role:            role,
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_remove_stale_knowledge_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "remove_stale_knowledge",
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_remove_stale_knowledge_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "remove_stale_knowledge",
			"result":    "failure",
		},
	})

	return dto.RemoveStaleKnowledgeOutput{}, err
}

func (uc *RemoveStaleKnowledgeUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	if uc.logger == nil {
		return nil
	}

	return uc.logger.Log(ctx, event)
}

func (uc *RemoveStaleKnowledgeUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *RemoveStaleKnowledgeUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, metric)
}
