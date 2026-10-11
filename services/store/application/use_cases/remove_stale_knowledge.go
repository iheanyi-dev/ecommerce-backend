package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

type RemoveStaleKnowledgeUseCase struct {
	storeRepository     ports.StoreRepository
	knowledgeRepository ports.KnowledgeRepository
	storage             ports.KnowledgeFileDeleter
	dispatcher          ports.KnowledgeDeletionDispatcher
	logger              ports.Logger
	metrics             ports.Metrics
}

func NewRemoveStaleKnowledgeUseCase(
	storeRepository ports.StoreRepository,
	knowledgeRepository ports.KnowledgeRepository,
	storage ports.KnowledgeFileDeleter,
	dispatcher ports.KnowledgeDeletionDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *RemoveStaleKnowledgeUseCase {
	return &RemoveStaleKnowledgeUseCase{
		storeRepository:     storeRepository,
		knowledgeRepository: knowledgeRepository,
		storage:             storage,
		dispatcher:          dispatcher,
		logger:              logger,
		metrics:             metrics,
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
		return uc.fail(ctx, startedAt, "authentication", applicationerrors.ErrUnauthenticated)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(ctx, startedAt, "repository_lookup", fmt.Errorf("find store: %w", err))
	}
	if store == nil || !store.IsOwnedBy(identity.UserID) {
		return uc.fail(ctx, startedAt, "authorization", applicationerrors.ErrStoreNotFound)
	}
	if !store.Plan().AIFeaturesAllowed() {
		return uc.fail(ctx, startedAt, "ai_not_allowed", fmt.Errorf("store plan does not allow AI features"))
	}
	if input.KnowledgeID == uuid.Nil {
		return uc.fail(ctx, startedAt, "knowledge_validation", fmt.Errorf("knowledge ID is required"))
	}

	knowledgeItems, err := uc.knowledgeRepository.GetByStoreID(ctx, store.ID().Value())
	if err != nil {
		return uc.fail(ctx, startedAt, "knowledge_lookup", fmt.Errorf("find store knowledge: %w", err))
	}

	var knowledgeReference string
	found := false
	for _, knowledge := range knowledgeItems {
		if knowledge != nil && knowledge.ID() == input.KnowledgeID {
			knowledgeReference = knowledge.StorageReference()
			found = true
			break
		}
	}
	if !found {
		return uc.fail(ctx, startedAt, "knowledge_not_found", applicationerrors.ErrStoreNotFound)
	}

	if err := uc.storage.DeleteKnowledgeFile(ctx, knowledgeReference); err != nil {
		return uc.fail(ctx, startedAt, "storage_deletion", fmt.Errorf("delete knowledge file from storage: %w", err))
	}
	if err := uc.knowledgeRepository.DeleteByID(ctx, input.KnowledgeID); err != nil {
		return uc.fail(ctx, startedAt, "metadata_deletion", fmt.Errorf("delete knowledge metadata: %w", err))
	}

	// Embedding deletion is best-effort after Storage and metadata deletion.
	if err := uc.dispatcher.DispatchKnowledgeDeletion(ctx, ports.KnowledgeDeletionRequested{
		KnowledgeID: input.KnowledgeID,
		StoreID:     store.ID().Value(),
	}); err != nil {
		_ = uc.log(ctx, ports.LogEvent{
			Event:           "store.remove_stale_knowledge.embedding_deletion_dispatch_failed",
			Operation:       "remove_stale_knowledge",
			UserID:          identity.UserID.String(),
			Role:            identity.Role,
			FailureCategory: "embedding_deletion_dispatch",
		})
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
		Name:   "store_remove_stale_knowledge_duration_seconds",
		Value:  time.Since(startedAt).Seconds(),
		Labels: map[string]string{"operation": "remove_stale_knowledge", "result": "success"},
	})

	return dto.RemoveStaleKnowledgeOutput{
		StoreID:     store.ID().Value(),
		KnowledgeID: input.KnowledgeID,
	}, nil
}

func (uc *RemoveStaleKnowledgeUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	category string,
	err error,
) (dto.RemoveStaleKnowledgeOutput, error) {
	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)
	userID, role := "", ""
	if authenticated {
		userID, role = identity.UserID.String(), identity.Role
	}
	_ = uc.log(ctx, ports.LogEvent{
		Event:           "store.remove_stale_knowledge.failed",
		Operation:       "remove_stale_knowledge",
		UserID:          userID,
		Role:            role,
		FailureCategory: category,
	})
	uc.increment(ctx, ports.Metric{
		Name:   "store_remove_stale_knowledge_failure_total",
		Value:  1,
		Labels: map[string]string{"operation": "remove_stale_knowledge", "failure_category": category},
	})
	uc.observe(ctx, ports.Metric{
		Name:   "store_remove_stale_knowledge_duration_seconds",
		Value:  time.Since(startedAt).Seconds(),
		Labels: map[string]string{"operation": "remove_stale_knowledge", "result": "failure"},
	})
	return dto.RemoveStaleKnowledgeOutput{}, err
}

func (uc *RemoveStaleKnowledgeUseCase) log(ctx context.Context, event ports.LogEvent) error {
	if uc.logger == nil {
		return nil
	}
	return uc.logger.Log(ctx, event)
}

func (uc *RemoveStaleKnowledgeUseCase) increment(ctx context.Context, metric ports.Metric) {
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, metric)
	}
}

func (uc *RemoveStaleKnowledgeUseCase) observe(ctx context.Context, metric ports.Metric) {
	if uc.metrics != nil {
		_ = uc.metrics.Observe(ctx, metric)
	}
}
