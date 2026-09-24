package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// AddStoreKnowledgeUseCase coordinates the submission of a Store knowledge
// file to the AI service.
//
// Store owns the authorization boundary and determines whether the Store's
// current plan allows AI features. The AI service owns knowledge processing,
// extraction, embeddings, and persistence of the resulting knowledge.
//
// The application layer therefore forwards the Store ID and file payload but
// does not know how the AI service stores or processes that knowledge.
type AddStoreKnowledgeUseCase struct {
	storeRepository     ports.StoreRepository
	knowledgeDispatcher ports.KnowledgeDispatcher
	logger              ports.Logger
	metrics             ports.Metrics
}

// NewAddStoreKnowledgeUseCase constructs the Add Store Knowledge use case.
func NewAddStoreKnowledgeUseCase(
	storeRepository ports.StoreRepository,
	knowledgeDispatcher ports.KnowledgeDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *AddStoreKnowledgeUseCase {
	return &AddStoreKnowledgeUseCase{
		storeRepository:     storeRepository,
		knowledgeDispatcher: knowledgeDispatcher,
		logger:              logger,
		metrics:             metrics,
	}
}

// Execute submits a Store knowledge file to the AI service.
//
// The authenticated identity is used only to authorize the Store owner. The
// user ID is deliberately not forwarded as part of Store knowledge because
// the knowledge belongs to the Store rather than to an individual user.
func (uc *AddStoreKnowledgeUseCase) Execute(
	ctx context.Context,
	input dto.AddStoreKnowledgeInput,
) (dto.AddStoreKnowledgeOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_add_knowledge_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "add_store_knowledge",
		},
	})

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"unauthenticated",
			applicationerrors.ErrUnauthenticated,
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_lookup",
			fmt.Errorf("find store: %w", err),
		)
	}

	if store == nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	// Knowledge management is authorized by Store ownership. As with the
	// other Store owner operations, a non-owner receives the same not-found
	// error so that Store existence is not unnecessarily disclosed.
	if !store.IsOwnedBy(identity.UserID) {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	// AI capability is determined by the Store's current plan. The Store
	// application layer does not need to know how the AI feature is
	// implemented; it only enforces the plan capability exposed by the
	// domain.
	if !store.Plan().AIFeaturesAllowed() {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"ai_not_allowed",
			fmt.Errorf("store plan does not allow AI features"),
		)
	}

	// Forward the file exactly as supplied. The AI service owns all knowledge
	// processing and persistence concerns.
	if err := uc.knowledgeDispatcher.DispatchStoreKnowledge(
		ctx,
		ports.StoreKnowledge{
			StoreID:     store.ID().Value(),
			Content:     input.File.Content,
			Filename:    input.File.Filename,
			ContentType: input.File.ContentType,
		},
	); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"knowledge_dispatch",
			fmt.Errorf("dispatch store knowledge: %w", err),
		)
	}

	output := dto.AddStoreKnowledgeOutput{
		StoreID: store.ID().Value(),
	}

	uc.log(ctx, ports.LogEvent{
		Event:     "store.add_knowledge.succeeded",
		Operation: "add_store_knowledge",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_add_knowledge_success_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "add_store_knowledge",
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_add_knowledge_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "add_store_knowledge",
			"result":    "success",
		},
	})

	return output, nil
}

// fail records failure observability while preserving the application error
// returned to the caller.
func (uc *AddStoreKnowledgeUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	identity ports.AuthenticatedIdentity,
	failureCategory string,
	err error,
) (dto.AddStoreKnowledgeOutput, error) {
	uc.log(ctx, ports.LogEvent{
		Event:           "store.add_knowledge.failed",
		Operation:       "add_store_knowledge",
		UserID:          identity.UserID.String(),
		Role:            identity.Role,
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_add_knowledge_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "add_store_knowledge",
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_add_knowledge_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "add_store_knowledge",
			"result":    "failure",
		},
	})

	return dto.AddStoreKnowledgeOutput{}, err
}

// log keeps logger failures from changing the outcome of the application
// operation.
func (uc *AddStoreKnowledgeUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

// increment keeps metrics failures from changing the outcome of the
// application operation.
func (uc *AddStoreKnowledgeUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

// observe keeps metrics failures from changing the outcome of the application
// operation.
func (uc *AddStoreKnowledgeUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, metric)
}
