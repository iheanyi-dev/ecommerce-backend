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
	storeRepo := &removeKnowledgeStoreRepo{}
	knowledgeRepo := &removeKnowledgeRepo{}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}

	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(context.Background(), dto.RemoveStaleKnowledgeInput{
		StoreID: uuid.New(), KnowledgeID: uuid.New(),
	})
	if !errors.Is(err, applicationerrors.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
	if storeRepo.findCalled || knowledgeRepo.getCalled || storage.called || dispatcher.called {
		t.Fatal("dependencies must not be called without authentication")
	}
}

func TestRemoveStaleKnowledgeUseCase_StoreNotFound(t *testing.T) {
	storeRepo := &removeKnowledgeStoreRepo{}
	knowledgeRepo := &removeKnowledgeRepo{}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	storeID := uuid.New()
	_, err := uc.Execute(removeKnowledgeContext(uuid.New()), dto.RemoveStaleKnowledgeInput{
		StoreID: storeID, KnowledgeID: uuid.New(),
	})
	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}
	if storeRepo.findID != storeID {
		t.Fatalf("expected store lookup %s, got %s", storeID, storeRepo.findID)
	}
	if knowledgeRepo.getCalled || storage.called || dispatcher.called {
		t.Fatal("deletion dependencies must not be called")
	}
}

func TestRemoveStaleKnowledgeUseCase_NonOwnerCannotDelete(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(uuid.New()), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: uuid.New(),
	})
	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}
	if knowledgeRepo.getCalled || storage.called || dispatcher.called {
		t.Fatal("deletion dependencies must not be called for a non-owner")
	}
}

func TestRemoveStaleKnowledgeUseCase_RejectsPlanWithoutAIFeatures(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypeBasic)
	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected AI capability error")
	}
	if knowledgeRepo.getCalled || storage.called || dispatcher.called {
		t.Fatal("deletion dependencies must not be called without AI features")
	}
}

func TestRemoveStaleKnowledgeUseCase_DeletesFileMetadataThenDispatches(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledge, err := entities.NewKnowledge(store.ID().Value(), "products.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("create knowledge: %v", err)
	}

	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{items: []*entities.Knowledge{knowledge}}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	output, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: knowledge.ID(),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if output.StoreID != store.ID().Value() || output.KnowledgeID != knowledge.ID() {
		t.Fatalf("unexpected output: %+v", output)
	}
	if !storage.called || storage.reference != knowledge.StorageReference() {
		t.Fatalf("expected storage deletion for %q; got called=%v reference=%q",
			knowledge.StorageReference(), storage.called, storage.reference)
	}
	if !knowledgeRepo.deleteCalled || knowledgeRepo.deletedID != knowledge.ID() {
		t.Fatalf("expected metadata deletion for %s", knowledge.ID())
	}
	if !dispatcher.called {
		t.Fatal("expected embedding deletion dispatch")
	}
	if dispatcher.request.KnowledgeID != knowledge.ID() ||
		dispatcher.request.StoreID != store.ID().Value() {
		t.Fatalf("unexpected dispatch request: %+v", dispatcher.request)
	}
}

func TestRemoveStaleKnowledgeUseCase_StorageFailurePreservesMetadata(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledge := removeKnowledgeTestKnowledge(t, store.ID().Value())
	expectedErr := errors.New("storage unavailable")

	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{items: []*entities.Knowledge{knowledge}}
	storage := &removeKnowledgeStorage{err: expectedErr}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: knowledge.ID(),
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected storage error, got %v", err)
	}
	if knowledgeRepo.deleteCalled || dispatcher.called {
		t.Fatal("metadata deletion and dispatch must stop after storage failure")
	}
}

func TestRemoveStaleKnowledgeUseCase_MetadataFailureDoesNotDispatch(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledge := removeKnowledgeTestKnowledge(t, store.ID().Value())
	expectedErr := errors.New("database unavailable")

	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{
		items: []*entities.Knowledge{knowledge}, deleteErr: expectedErr,
	}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: knowledge.ID(),
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected metadata deletion error, got %v", err)
	}
	if !storage.called {
		t.Fatal("expected storage deletion before metadata deletion")
	}
	if dispatcher.called {
		t.Fatal("must not dispatch when metadata deletion fails")
	}
}

func TestRemoveStaleKnowledgeUseCase_DispatchFailureIsBestEffort(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledge := removeKnowledgeTestKnowledge(t, store.ID().Value())

	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{items: []*entities.Knowledge{knowledge}}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{err: errors.New("Kafka unavailable")}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: knowledge.ID(),
	})
	if err != nil {
		t.Fatalf("dispatch failure must not fail deletion: %v", err)
	}
	if !knowledgeRepo.deleteCalled || !dispatcher.called {
		t.Fatal("expected metadata deletion and dispatch attempt")
	}
}

func TestRemoveStaleKnowledgeUseCase_KnowledgeNotFound(t *testing.T) {
	ownerID := uuid.New()
	store := removeKnowledgeTestStore(t, ownerID, valueobjects.PlanTypePremium)
	storeRepo := &removeKnowledgeStoreRepo{store: store}
	knowledgeRepo := &removeKnowledgeRepo{}
	storage := &removeKnowledgeStorage{}
	dispatcher := &removeKnowledgeDispatcher{}
	uc := newRemoveKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher)

	_, err := uc.Execute(removeKnowledgeContext(ownerID), dto.RemoveStaleKnowledgeInput{
		StoreID: store.ID().Value(), KnowledgeID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected an error for unknown knowledge")
	}
	if storage.called || knowledgeRepo.deleteCalled || dispatcher.called {
		t.Fatal("must not delete or dispatch unknown knowledge")
	}
}

func newRemoveKnowledgeUseCase(
	storeRepo *removeKnowledgeStoreRepo,
	knowledgeRepo *removeKnowledgeRepo,
	storage *removeKnowledgeStorage,
	dispatcher *removeKnowledgeDispatcher,
) *usecases.RemoveStaleKnowledgeUseCase {
	return usecases.NewRemoveStaleKnowledgeUseCase(
		storeRepo, knowledgeRepo, storage, dispatcher,
		&removeKnowledgeLogger{}, &removeKnowledgeMetrics{},
	)
}

type removeKnowledgeStoreRepo struct {
	store      *entities.Store
	findCalled bool
	findID     uuid.UUID
}

func (f *removeKnowledgeStoreRepo) Create(context.Context, *entities.Store) error { return nil }
func (f *removeKnowledgeStoreRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f *removeKnowledgeStoreRepo) FindByID(_ context.Context, id uuid.UUID) (*entities.Store, error) {
	f.findCalled, f.findID = true, id
	return f.store, nil
}
func (f *removeKnowledgeStoreRepo) FindByOwnerID(context.Context, uuid.UUID) (*entities.Store, error) {
	return nil, nil
}
func (f *removeKnowledgeStoreRepo) FindBySlug(context.Context, string) (*entities.Store, error) {
	return nil, nil
}
func (f *removeKnowledgeStoreRepo) ExistsBySlug(context.Context, string) (bool, error) {
	return false, nil
}
func (f *removeKnowledgeStoreRepo) ListActive(context.Context, string, int, int) ([]*entities.Store, int, error) {
	return nil, 0, nil
}
func (f *removeKnowledgeStoreRepo) Update(context.Context, *entities.Store) error     { return nil }
func (f *removeKnowledgeStoreRepo) ChangePlan(context.Context, *entities.Store) error { return nil }
func (f *removeKnowledgeStoreRepo) ChangeStatus(context.Context, uuid.UUID, string) error {
	return nil
}

type removeKnowledgeRepo struct {
	items        []*entities.Knowledge
	getCalled    bool
	deleteCalled bool
	deletedID    uuid.UUID
	deleteErr    error
}

func (f *removeKnowledgeRepo) GetByStoreID(context.Context, uuid.UUID) ([]*entities.Knowledge, error) {
	f.getCalled = true
	return f.items, nil
}
func (f *removeKnowledgeRepo) Save(context.Context, *entities.Knowledge) error { return nil }
func (f *removeKnowledgeRepo) DeleteByID(_ context.Context, id uuid.UUID) error {
	f.deleteCalled, f.deletedID = true, id
	return f.deleteErr
}

type removeKnowledgeStorage struct {
	called    bool
	reference string
	err       error
}

func (f *removeKnowledgeStorage) DeleteKnowledgeFile(_ context.Context, reference string) error {
	f.called, f.reference = true, reference
	return f.err
}

type removeKnowledgeDispatcher struct {
	called  bool
	request ports.KnowledgeDeletionRequested
	err     error
}

func (f *removeKnowledgeDispatcher) DispatchKnowledgeDeletion(
	_ context.Context,
	request ports.KnowledgeDeletionRequested,
) error {
	f.called, f.request = true, request
	return f.err
}

type removeKnowledgeLogger struct{}

func (*removeKnowledgeLogger) Log(context.Context, ports.LogEvent) error { return nil }

type removeKnowledgeMetrics struct{}

func (*removeKnowledgeMetrics) Increment(context.Context, ports.Metric) error { return nil }
func (*removeKnowledgeMetrics) Observe(context.Context, ports.Metric) error   { return nil }

func removeKnowledgeTestStore(
	t *testing.T,
	ownerID uuid.UUID,
	planType valueobjects.PlanType,
) *entities.Store {
	t.Helper()
	store, err := entities.NewStore(
		ownerID, "Knowledge Store", "knowledge-store", "Knowledge test store",
		nil, string(planType),
	)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store
}

func removeKnowledgeTestKnowledge(t *testing.T, storeID uuid.UUID) *entities.Knowledge {
	t.Helper()
	knowledge, err := entities.NewKnowledge(storeID, "products.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("create knowledge: %v", err)
	}
	return knowledge
}

func removeKnowledgeContext(userID uuid.UUID) context.Context {
	return ports.WithAuthenticatedIdentity(context.Background(), ports.AuthenticatedIdentity{
		UserID: userID,
		Role:   "vendor",
	})
}
