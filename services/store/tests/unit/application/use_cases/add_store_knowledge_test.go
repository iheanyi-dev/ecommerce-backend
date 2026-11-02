package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

func TestAddStoreKnowledgeUseCase_RequiresAuthentication(t *testing.T) {
	storeRepo := &addKnowledgeStoreRepository{}
	knowledgeRepo := &addKnowledgeRepository{}
	storage := &addKnowledgeStorage{}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher, &addKnowledgeLogger{}, &addKnowledgeMetrics{})

	_, err := uc.Execute(context.Background(), addKnowledgeInput(uuid.New()))
	if !errors.Is(err, applicationerrors.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
	if storeRepo.findByIDCalled || storage.called || knowledgeRepo.saveCalled || dispatcher.called {
		t.Fatal("dependencies should not be called for unauthenticated requests")
	}
}

func TestAddStoreKnowledgeUseCase_StoreNotFound(t *testing.T) {
	storeRepo := &addKnowledgeStoreRepository{}
	storage := &addKnowledgeStorage{}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(storeRepo, &addKnowledgeRepository{}, storage, dispatcher, &addKnowledgeLogger{}, &addKnowledgeMetrics{})

	_, err := uc.Execute(addKnowledgeAuthenticatedContext(context.Background(), uuid.New()), addKnowledgeInput(uuid.New()))
	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}
	if storage.called || dispatcher.called {
		t.Fatal("storage and dispatcher must not be called when store is missing")
	}
}

func TestAddStoreKnowledgeUseCase_NonOwnerCannotAddKnowledge(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)
	storage := &addKnowledgeStorage{}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		&addKnowledgeRepository{},
		storage,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), uuid.New()),
		addKnowledgeInput(store.ID().Value()),
	)
	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}
	if storage.called || dispatcher.called {
		t.Fatal("storage and dispatcher must not be called for a non-owner")
	}
}

func TestAddStoreKnowledgeUseCase_RejectsBasicPlan(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypeBasic)
	storage := &addKnowledgeStorage{}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		&addKnowledgeRepository{},
		storage,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), ownerID),
		addKnowledgeInput(store.ID().Value()),
	)
	if err == nil {
		t.Fatal("expected AI capability error")
	}
	if storage.called || dispatcher.called {
		t.Fatal("storage and dispatcher must not be called for a plan without AI")
	}
}

func TestAddStoreKnowledgeUseCase_StorageFailureDoesNotSaveMetadata(t *testing.T) {
	expectedErr := errors.New("storage unavailable")
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledgeRepo := &addKnowledgeRepository{}
	storage := &addKnowledgeStorage{err: expectedErr}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		knowledgeRepo,
		storage,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), ownerID),
		addKnowledgeInput(store.ID().Value()),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected storage error, got %v", err)
	}
	if knowledgeRepo.saveCalled || dispatcher.called {
		t.Fatal("metadata persistence and ingestion must not happen after storage failure")
	}
}

func TestAddStoreKnowledgeUseCase_MetadataFailureDoesNotDispatch(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)
	storage := &addKnowledgeStorage{}
	knowledgeRepo := &addKnowledgeRepository{saveErr: expectedErr}
	dispatcher := &addKnowledgeDispatcher{}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		knowledgeRepo,
		storage,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), ownerID),
		addKnowledgeInput(store.ID().Value()),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected metadata persistence error, got %v", err)
	}
	if !storage.called {
		t.Fatal("expected file to be uploaded before metadata persistence")
	}
	if dispatcher.called {
		t.Fatal("ingestion must not be dispatched when metadata persistence fails")
	}
}

func TestAddStoreKnowledgeUseCase_PersistsAndDispatchesKnowledgeID(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledgeRepo := &addKnowledgeRepository{}
	storage := &addKnowledgeStorage{}
	dispatcher := &addKnowledgeDispatcher{}
	logger := &addKnowledgeLogger{}
	metrics := &addKnowledgeMetrics{}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		knowledgeRepo,
		storage,
		dispatcher,
		logger,
		metrics,
	)

	content := []byte("enterprise product documentation")
	input := dto.AddStoreKnowledgeInput{
		StoreID: store.ID().Value(),
		File: dto.StoreKnowledgeFileInput{
			Content: content, Filename: "products.pdf", ContentType: "application/pdf",
		},
	}
	output, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), ownerID),
		input,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if output.StoreID != store.ID().Value() {
		t.Fatalf("expected StoreID %s, got %s", store.ID().Value(), output.StoreID)
	}
	if !storage.called {
		t.Fatal("expected file upload")
	}
	if storage.file.Reference != "knowledge/"+store.ID().Value().String()+"/products.pdf" {
		t.Fatalf("unexpected storage reference: %q", storage.file.Reference)
	}
	if string(storage.file.Content) != string(content) {
		t.Fatal("uploaded content does not match input")
	}
	if storage.file.ContentType != "application/pdf" {
		t.Fatalf("unexpected content type: %q", storage.file.ContentType)
	}
	if !knowledgeRepo.saveCalled || knowledgeRepo.saved == nil {
		t.Fatal("expected knowledge metadata to be persisted")
	}
	if knowledgeRepo.saved.StoreID() != store.ID().Value() ||
		knowledgeRepo.saved.Filename() != "products.pdf" ||
		knowledgeRepo.saved.ContentType() != "application/pdf" ||
		knowledgeRepo.saved.StorageReference() != storage.file.Reference {
		t.Fatal("persisted knowledge metadata does not match uploaded file")
	}
	if !dispatcher.called {
		t.Fatal("expected ingestion dispatch")
	}
	if dispatcher.request.KnowledgeID != knowledgeRepo.saved.ID() {
		t.Fatalf("expected dispatched knowledge ID %s, got %s", knowledgeRepo.saved.ID(), dispatcher.request.KnowledgeID)
	}
	if dispatcher.request.StoreID != store.ID().Value() {
		t.Fatalf("unexpected dispatched store ID: %s", dispatcher.request.StoreID)
	}
	if dispatcher.request.StorageReference != storage.file.Reference {
		t.Fatalf("unexpected dispatched storage reference: %q", dispatcher.request.StorageReference)
	}
	if !logger.successCalled {
		t.Fatal("expected success log")
	}
	if metrics.incrementCalls == 0 || metrics.observeCalls == 0 {
		t.Fatal("expected success metrics")
	}
}

func TestAddStoreKnowledgeUseCase_KafkaFailureIsNonFatal(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)
	knowledgeRepo := &addKnowledgeRepository{}
	dispatcher := &addKnowledgeDispatcher{err: errors.New("kafka unavailable")}
	uc := newAddKnowledgeUseCase(
		&addKnowledgeStoreRepository{store: store},
		knowledgeRepo,
		&addKnowledgeStorage{},
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	_, err := uc.Execute(
		addKnowledgeAuthenticatedContext(context.Background(), ownerID),
		addKnowledgeInput(store.ID().Value()),
	)
	if err != nil {
		t.Fatalf("expected successful upload despite Kafka failure, got %v", err)
	}
	if !knowledgeRepo.saveCalled {
		t.Fatal("expected metadata to remain persisted")
	}
	if !dispatcher.called {
		t.Fatal("expected ingestion dispatch attempt")
	}
}

func addKnowledgeInput(storeID uuid.UUID) dto.AddStoreKnowledgeInput {
	return dto.AddStoreKnowledgeInput{
		StoreID: storeID,
		File: dto.StoreKnowledgeFileInput{
			Content: []byte("knowledge"), Filename: "knowledge.pdf", ContentType: "application/pdf",
		},
	}
}

func newAddKnowledgeUseCase(
	storeRepo ports.StoreRepository,
	knowledgeRepo ports.KnowledgeRepository,
	storage ports.KnowledgeStorage,
	dispatcher ports.KnowledgeIngestionDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *usecases.AddStoreKnowledgeUseCase {
	return usecases.NewAddStoreKnowledgeUseCase(storeRepo, knowledgeRepo, storage, dispatcher, logger, metrics)
}

type addKnowledgeStoreRepository struct {
	store          *entities.Store
	findByIDError  error
	findByIDCalled bool
	findByIDID     uuid.UUID
}

func (f *addKnowledgeStoreRepository) Create(context.Context, *entities.Store) error { return nil }
func (f *addKnowledgeStoreRepository) Delete(context.Context, uuid.UUID) error       { return nil }
func (f *addKnowledgeStoreRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Store, error) {
	f.findByIDCalled, f.findByIDID = true, id
	if f.findByIDError != nil {
		return nil, f.findByIDError
	}
	return f.store, nil
}
func (f *addKnowledgeStoreRepository) FindByOwnerID(context.Context, uuid.UUID) (*entities.Store, error) {
	return nil, nil
}
func (f *addKnowledgeStoreRepository) FindBySlug(context.Context, string) (*entities.Store, error) {
	return nil, nil
}
func (f *addKnowledgeStoreRepository) ExistsBySlug(context.Context, string) (bool, error) {
	return false, nil
}
func (f *addKnowledgeStoreRepository) ListActive(context.Context, string, int, int) ([]*entities.Store, int, error) {
	return nil, 0, nil
}
func (f *addKnowledgeStoreRepository) Update(context.Context, *entities.Store) error     { return nil }
func (f *addKnowledgeStoreRepository) ChangePlan(context.Context, *entities.Store) error { return nil }
func (f *addKnowledgeStoreRepository) ChangeStatus(context.Context, uuid.UUID, string) error {
	return nil
}

type addKnowledgeRepository struct {
	saved      *entities.Knowledge
	saveCalled bool
	saveErr    error
}

func (f *addKnowledgeRepository) GetByStoreID(context.Context, uuid.UUID) ([]*entities.Knowledge, error) {
	return nil, nil
}
func (f *addKnowledgeRepository) Save(_ context.Context, knowledge *entities.Knowledge) error {
	f.saveCalled = true
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = knowledge
	return nil
}
func (f *addKnowledgeRepository) DeleteByID(context.Context, uuid.UUID) error { return nil }

type addKnowledgeStorage struct {
	called bool
	file   ports.KnowledgeFile
	err    error
}

func (f *addKnowledgeStorage) PutKnowledgeFile(_ context.Context, file ports.KnowledgeFile) error {
	f.called, f.file = true, file
	return f.err
}

type addKnowledgeDispatcher struct {
	called  bool
	request ports.KnowledgeIngestionRequested
	err     error
}

func (f *addKnowledgeDispatcher) DispatchKnowledgeIngestion(_ context.Context, request ports.KnowledgeIngestionRequested) error {
	f.called, f.request = true, request
	return f.err
}

type addKnowledgeLogger struct{ successCalled bool }

func (f *addKnowledgeLogger) Log(_ context.Context, event ports.LogEvent) error {
	if event.Event == "store.add_knowledge.succeeded" {
		f.successCalled = true
	}
	return nil
}

type addKnowledgeMetrics struct{ incrementCalls, observeCalls int }

func (f *addKnowledgeMetrics) Increment(context.Context, ports.Metric) error {
	f.incrementCalls++
	return nil
}
func (f *addKnowledgeMetrics) Observe(context.Context, ports.Metric) error {
	f.observeCalls++
	return nil
}

func newAddKnowledgeStore(
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

func addKnowledgeAuthenticatedContext(
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

// Keep time imported as part of this test's explicit dependency set while the
// observability fakes remain intentionally minimal.
var _ = time.Second
