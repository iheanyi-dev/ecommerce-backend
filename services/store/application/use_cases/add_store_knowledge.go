package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
)

// AddStoreKnowledgeUseCase stores a Knowledge file and its metadata before
// publishing a best-effort ingestion request for the Embedding service.
type AddStoreKnowledgeUseCase struct {
	storeRepository     ports.StoreRepository
	knowledgeRepository ports.KnowledgeRepository
	knowledgeStorage    ports.KnowledgeStorage
	ingestionDispatcher ports.KnowledgeIngestionDispatcher
	logger              ports.Logger
	metrics             ports.Metrics
}

func NewAddStoreKnowledgeUseCase(
	storeRepository ports.StoreRepository,
	knowledgeRepository ports.KnowledgeRepository,
	knowledgeStorage ports.KnowledgeStorage,
	ingestionDispatcher ports.KnowledgeIngestionDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *AddStoreKnowledgeUseCase {
	return &AddStoreKnowledgeUseCase{
		storeRepository:     storeRepository,
		knowledgeRepository: knowledgeRepository,
		knowledgeStorage:    knowledgeStorage,
		ingestionDispatcher: ingestionDispatcher,
		logger:              logger,
		metrics:             metrics,
	}
}

func (uc *AddStoreKnowledgeUseCase) Execute(
	ctx context.Context,
	input dto.AddStoreKnowledgeInput,
) (dto.AddStoreKnowledgeOutput, error) {
	startedAt := time.Now()
	uc.increment(ctx, ports.Metric{
		Name: "store_add_knowledge_total", Value: 1,
		Labels: map[string]string{"operation": "add_store_knowledge"},
	})

	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)
	if !authenticated || identity.UserID == uuid.Nil {
		return uc.fail(ctx, startedAt, identity, "unauthenticated", applicationerrors.ErrUnauthenticated)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(ctx, startedAt, identity, "store_lookup", fmt.Errorf("find store: %w", err))
	}
	if store == nil || !store.IsOwnedBy(identity.UserID) {
		return uc.fail(ctx, startedAt, identity, "store_not_found", applicationerrors.ErrStoreNotFound)
	}
	if !store.Plan().AIFeaturesAllowed() {
		return uc.fail(ctx, startedAt, identity, "ai_not_allowed", fmt.Errorf("store plan does not allow AI features"))
	}

	knowledge, err := entities.NewKnowledge(
		store.ID().Value(),
		input.File.Filename,
		input.File.ContentType,
	)
	if err != nil {
		return uc.fail(ctx, startedAt, identity, "invalid_knowledge", fmt.Errorf("create knowledge metadata: %w", err))
	}

	// Storage is synchronous: metadata must not be saved if the upload fails.
	if err := uc.knowledgeStorage.PutKnowledgeFile(ctx, ports.KnowledgeFile{
		Reference:   knowledge.StorageReference(),
		Content:     input.File.Content,
		ContentType: input.File.ContentType,
	}); err != nil {
		return uc.fail(ctx, startedAt, identity, "storage_upload", fmt.Errorf("upload knowledge file: %w", err))
	}

	// Persist the metadata before notifying the asynchronous ingestion worker.
	if err := uc.knowledgeRepository.Save(ctx, knowledge); err != nil {
		return uc.fail(ctx, startedAt, identity, "knowledge_persistence", fmt.Errorf("save knowledge metadata: %w", err))
	}

	// Publishing is best-effort by design. A broker outage must not report
	// failure after Storage and the metadata repository have accepted the file.
	if err := uc.ingestionDispatcher.DispatchKnowledgeIngestion(ctx, ports.KnowledgeIngestionRequested{
		KnowledgeID:      knowledge.ID(),
		StoreID:          knowledge.StoreID(),
		StorageReference: knowledge.StorageReference(),
	}); err != nil {
		uc.log(ctx, ports.LogEvent{
			Event:           "store.add_knowledge.ingestion_dispatch_failed",
			Operation:       "add_store_knowledge",
			UserID:          identity.UserID.String(),
			Role:            identity.Role,
			FailureCategory: "knowledge_dispatch",
		})
	}

	uc.log(ctx, ports.LogEvent{
		Event:     "store.add_knowledge.succeeded",
		Operation: "add_store_knowledge",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})
	uc.increment(ctx, ports.Metric{
		Name: "store_add_knowledge_success_total", Value: 1,
		Labels: map[string]string{"operation": "add_store_knowledge"},
	})
	uc.observe(ctx, ports.Metric{
		Name:   "store_add_knowledge_duration_seconds",
		Value:  time.Since(startedAt).Seconds(),
		Labels: map[string]string{"operation": "add_store_knowledge", "result": "success"},
	})

	return dto.AddStoreKnowledgeOutput{StoreID: store.ID().Value()}, nil
}

func (uc *AddStoreKnowledgeUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	identity ports.AuthenticatedIdentity,
	category string,
	err error,
) (dto.AddStoreKnowledgeOutput, error) {
	userID := ""
	role := ""
	if identity.UserID != uuid.Nil {
		userID = identity.UserID.String()
		role = identity.Role
	}

	uc.log(ctx, ports.LogEvent{
		Event:           "store.add_knowledge.failed",
		Operation:       "add_store_knowledge",
		UserID:          userID,
		Role:            role,
		FailureCategory: category,
	})
	uc.increment(ctx, ports.Metric{
		Name: "store_add_knowledge_failure_total", Value: 1,
		Labels: map[string]string{"operation": "add_store_knowledge", "failure_category": category},
	})
	uc.observe(ctx, ports.Metric{
		Name:   "store_add_knowledge_duration_seconds",
		Value:  time.Since(startedAt).Seconds(),
		Labels: map[string]string{"operation": "add_store_knowledge", "result": "failure"},
	})
	return dto.AddStoreKnowledgeOutput{}, err
}

func (uc *AddStoreKnowledgeUseCase) log(ctx context.Context, event ports.LogEvent) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, event)
	}
}

func (uc *AddStoreKnowledgeUseCase) increment(ctx context.Context, metric ports.Metric) {
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, metric)
	}
}

func (uc *AddStoreKnowledgeUseCase) observe(ctx context.Context, metric ports.Metric) {
	if uc.metrics != nil {
		_ = uc.metrics.Observe(ctx, metric)
	}
}
