package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

func TestRemoveStaleKnowledgeUseCase_RequiresAuthentication(t *testing.T) {
	repository := &removeStaleKnowledgeStoreRepository{}
	dispatcher := &removeStaleKnowledgeDispatcher{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.RemoveStaleKnowledgeInput{
			StoreID: uuid.New(),
		},
	)

	if !errors.Is(err, applicationerrors.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}

	if repository.findByIDCalled {
		t.Fatal("expected repository not to be called")
	}

	if dispatcher.called {
		t.Fatal("expected dispatcher not to be called")
	}
}

func TestRemoveStaleKnowledgeUseCase_StoreNotFound(t *testing.T) {
	repository := &removeStaleKnowledgeStoreRepository{}
	dispatcher := &removeStaleKnowledgeDispatcher{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	userID := uuid.New()
	storeID := uuid.New()

	_, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), userID),
		dto.RemoveStaleKnowledgeInput{
			StoreID: storeID,
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if repository.findByIDID != storeID {
		t.Fatalf("expected lookup for %s, got %s", storeID, repository.findByIDID)
	}

	if dispatcher.called {
		t.Fatal("expected dispatcher not to be called")
	}
}

func TestRemoveStaleKnowledgeUseCase_NonOwnerReceivesStoreNotFound(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()

	store := newRemoveStaleKnowledgeStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &removeStaleKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &removeStaleKnowledgeDispatcher{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), otherUserID),
		dto.RemoveStaleKnowledgeInput{
			StoreID: store.ID().Value(),
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if dispatcher.called {
		t.Fatal("expected dispatcher not to be called")
	}
}

func TestRemoveStaleKnowledgeUseCase_RejectsStoreWithoutAIFeatures(t *testing.T) {
	ownerID := uuid.New()

	store := newRemoveStaleKnowledgeStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
	)

	repository := &removeStaleKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &removeStaleKnowledgeDispatcher{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), ownerID),
		dto.RemoveStaleKnowledgeInput{
			StoreID: store.ID().Value(),
		},
	)

	if err == nil {
		t.Fatal("expected AI capability error")
	}

	if dispatcher.called {
		t.Fatal("expected dispatcher not to be called")
	}
}

func TestRemoveStaleKnowledgeUseCase_DispatchesStoreIDAndFilename(t *testing.T) {
	ownerID := uuid.New()

	store := newRemoveStaleKnowledgeStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &removeStaleKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &removeStaleKnowledgeDispatcher{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	output, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), ownerID),
		dto.RemoveStaleKnowledgeInput{
			StoreID:  store.ID().Value(),
			Filename: "products.pdf",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StoreID != store.ID().Value() {
		t.Fatalf(
			"expected output StoreID %s, got %s",
			store.ID().Value(),
			output.StoreID,
		)
	}

	if !dispatcher.called {
		t.Fatal("expected dispatcher to be called")
	}

	if dispatcher.knowledge.StoreID != store.ID().Value() {
		t.Fatalf(
			"expected dispatched StoreID %s, got %s",
			store.ID().Value(),
			dispatcher.knowledge.StoreID,
		)
	}

	if dispatcher.knowledge.Filename != "products.pdf" {
		t.Fatalf(
			"expected dispatched filename %q, got %q",
			"products.pdf",
			dispatcher.knowledge.Filename,
		)
	}

	if output.Filename != "products.pdf" {
		t.Fatalf(
			"expected output filename %q, got %q",
			"products.pdf",
			output.Filename,
		)
	}
}

func TestRemoveStaleKnowledgeUseCase_DispatchFailureIsReturned(t *testing.T) {
	expectedErr := errors.New("AI service unavailable")

	ownerID := uuid.New()

	store := newRemoveStaleKnowledgeStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &removeStaleKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &removeStaleKnowledgeDispatcher{
		dispatchError: expectedErr,
	}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		repository,
		dispatcher,
		&removeStaleKnowledgeLogger{},
		&removeStaleKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), ownerID),
		dto.RemoveStaleKnowledgeInput{
			StoreID: store.ID().Value(),
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected dispatcher error to be returned, got %v", err)
	}
}

func TestRemoveStaleKnowledgeUseCase_ObservesSuccessfulOperation(t *testing.T) {
	ownerID := uuid.New()

	store := newRemoveStaleKnowledgeStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	logger := &removeStaleKnowledgeLogger{}
	metrics := &removeStaleKnowledgeMetrics{}

	uc := usecases.NewRemoveStaleKnowledgeUseCase(
		&removeStaleKnowledgeStoreRepository{store: store},
		&removeStaleKnowledgeDispatcher{},
		logger,
		metrics,
	)

	_, err := uc.Execute(
		removeStaleKnowledgeAuthenticatedContext(context.Background(), ownerID),
		dto.RemoveStaleKnowledgeInput{
			StoreID: store.ID().Value(),
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !logger.successCalled {
		t.Fatal("expected success log")
	}

	if metrics.incrementCalls == 0 {
		t.Fatal("expected metrics increment")
	}

	if metrics.observeCalls == 0 {
		t.Fatal("expected metrics observation")
	}
}

type removeStaleKnowledgeStoreRepository struct {
	store          *entities.Store
	findByIDCalled bool
	findByIDID     uuid.UUID
}

func (f *removeStaleKnowledgeStoreRepository) Create(context.Context, *entities.Store) error {
	return nil
}

func (f *removeStaleKnowledgeStoreRepository) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (f *removeStaleKnowledgeStoreRepository) FindByID(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true
	f.findByIDID = storeID
	return f.store, nil
}

func (f *removeStaleKnowledgeStoreRepository) FindByOwnerID(context.Context, uuid.UUID) (*entities.Store, error) {
	return nil, nil
}

func (f *removeStaleKnowledgeStoreRepository) FindBySlug(context.Context, string) (*entities.Store, error) {
	return nil, nil
}

func (f *removeStaleKnowledgeStoreRepository) ExistsBySlug(context.Context, string) (bool, error) {
	return false, nil
}

func (f *removeStaleKnowledgeStoreRepository) ListActive(
	context.Context,
	string,
	int,
	int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *removeStaleKnowledgeStoreRepository) Update(context.Context, *entities.Store) error {
	return nil
}

func (f *removeStaleKnowledgeStoreRepository) ChangePlan(context.Context, *entities.Store) error {
	return nil
}

func (f *removeStaleKnowledgeStoreRepository) ChangeStatus(
	context.Context,
	uuid.UUID,
	string,
) error {
	return nil
}

type removeStaleKnowledgeDispatcher struct {
	called        bool
	knowledge     ports.StaleKnowledge
	dispatchError error
}

func (f *removeStaleKnowledgeDispatcher) DispatchRemoveStaleKnowledge(
	_ context.Context,
	knowledge ports.StaleKnowledge,
) error {
	f.called = true
	f.knowledge = knowledge
	return f.dispatchError
}

type removeStaleKnowledgeLogger struct {
	successCalled bool
}

func (f *removeStaleKnowledgeLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "store.remove_stale_knowledge.succeeded" {
		f.successCalled = true
	}
	return nil
}

type removeStaleKnowledgeMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *removeStaleKnowledgeMetrics) Increment(context.Context, ports.Metric) error {
	f.incrementCalls++
	return nil
}

func (f *removeStaleKnowledgeMetrics) Observe(context.Context, ports.Metric) error {
	f.observeCalls++
	return nil
}

func newRemoveStaleKnowledgeStore(
	t *testing.T,
	ownerID uuid.UUID,
	planType valueobjects.PlanType,
) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		ownerID,
		"Knowledge Store",
		"knowledge-store",
		"Knowledge test store",
		nil,
		string(planType),
	)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	return store
}

func removeStaleKnowledgeAuthenticatedContext(
	ctx context.Context,
	userID uuid.UUID,
) context.Context {
	return ports.WithAuthenticatedIdentity(
		ctx,
		ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   "vendor",
		},
	)
}
