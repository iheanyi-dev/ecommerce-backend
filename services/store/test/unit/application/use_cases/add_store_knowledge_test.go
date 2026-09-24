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
	repository := &addKnowledgeStoreRepository{}
	dispatcher := &addKnowledgeDispatcher{}
	logger := &addKnowledgeLogger{}
	metrics := &addKnowledgeMetrics{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		context.Background(),
		dto.AddStoreKnowledgeInput{
			StoreID: uuid.New(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if !errors.Is(err, applicationerrors.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}

	if repository.findByIDCalled {
		t.Fatal("expected repository not to be called")
	}

	if dispatcher.called {
		t.Fatal("expected knowledge dispatcher not to be called")
	}
}

func TestAddStoreKnowledgeUseCase_StoreNotFound(t *testing.T) {
	repository := &addKnowledgeStoreRepository{}
	dispatcher := &addKnowledgeDispatcher{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	storeID := uuid.New()
	ownerID := uuid.New()

	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: storeID,
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if repository.findByIDID != storeID {
		t.Fatalf("expected repository lookup for %s, got %s", storeID, repository.findByIDID)
	}

	if dispatcher.called {
		t.Fatal("expected knowledge dispatcher not to be called")
	}
}

func TestAddStoreKnowledgeUseCase_RepositoryLookupFailure(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	repository := &addKnowledgeStoreRepository{
		findByIDError: expectedErr,
	}
	dispatcher := &addKnowledgeDispatcher{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	ownerID := uuid.New()
	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: uuid.New(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error to be wrapped, got %v", err)
	}

	if dispatcher.called {
		t.Fatal("expected knowledge dispatcher not to be called")
	}
}

func TestAddStoreKnowledgeUseCase_NonOwnerReceivesStoreNotFound(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)

	repository := &addKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &addKnowledgeDispatcher{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	ctx := addKnowledgeAuthenticatedContext(context.Background(), otherUserID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: store.ID().Value(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if dispatcher.called {
		t.Fatal("expected knowledge dispatcher not to be called")
	}
}

func TestAddStoreKnowledgeUseCase_RejectsStoreWithoutAIFeatures(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypeBasic)

	repository := &addKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &addKnowledgeDispatcher{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: store.ID().Value(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if err == nil {
		t.Fatal("expected AI capability error")
	}

	if dispatcher.called {
		t.Fatal("expected knowledge dispatcher not to be called")
	}
}

func TestAddStoreKnowledgeUseCase_DispatchesStoreIDAndFile(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)

	repository := &addKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &addKnowledgeDispatcher{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	content := []byte("enterprise product documentation")

	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	input := dto.AddStoreKnowledgeInput{
		StoreID: store.ID().Value(),
		File: dto.StoreKnowledgeFileInput{
			Content:     content,
			Filename:    "products.pdf",
			ContentType: "application/pdf",
		},
	}

	output, err := uc.Execute(ctx, input)
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
		t.Fatal("expected knowledge dispatcher to be called")
	}

	if dispatcher.knowledge.StoreID != store.ID().Value() {
		t.Fatalf(
			"expected dispatched StoreID %s, got %s",
			store.ID().Value(),
			dispatcher.knowledge.StoreID,
		)
	}

	if string(dispatcher.knowledge.Content) != string(content) {
		t.Fatalf("expected dispatched content %q, got %q", content, dispatcher.knowledge.Content)
	}

	if dispatcher.knowledge.Filename != "products.pdf" {
		t.Fatalf(
			"expected filename %q, got %q",
			"products.pdf",
			dispatcher.knowledge.Filename,
		)
	}

	if dispatcher.knowledge.ContentType != "application/pdf" {
		t.Fatalf(
			"expected content type %q, got %q",
			"application/pdf",
			dispatcher.knowledge.ContentType,
		)
	}
}

func TestAddStoreKnowledgeUseCase_DispatchFailureIsReturned(t *testing.T) {
	expectedErr := errors.New("AI service unavailable")

	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)

	repository := &addKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &addKnowledgeDispatcher{
		dispatchError: expectedErr,
	}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		&addKnowledgeLogger{},
		&addKnowledgeMetrics{},
	)

	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: store.ID().Value(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected dispatcher error to be returned, got %v", err)
	}
}

func TestAddStoreKnowledgeUseCase_ObservesSuccessfulOperation(t *testing.T) {
	ownerID := uuid.New()
	store := newAddKnowledgeStore(t, ownerID, valueobjects.PlanTypePremium)

	repository := &addKnowledgeStoreRepository{
		store: store,
	}
	dispatcher := &addKnowledgeDispatcher{}
	logger := &addKnowledgeLogger{}
	metrics := &addKnowledgeMetrics{}

	uc := usecases.NewAddStoreKnowledgeUseCase(
		repository,
		dispatcher,
		logger,
		metrics,
	)

	ctx := addKnowledgeAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.AddStoreKnowledgeInput{
			StoreID: store.ID().Value(),
			File: dto.StoreKnowledgeFileInput{
				Content:     []byte("knowledge"),
				Filename:    "knowledge.pdf",
				ContentType: "application/pdf",
			},
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

type addKnowledgeStoreRepository struct {
	store          *entities.Store
	findByIDError  error
	findByIDCalled bool
	findByIDID     uuid.UUID
}

func (f *addKnowledgeStoreRepository) Create(context.Context, *entities.Store) error {
	return nil
}

func (f *addKnowledgeStoreRepository) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (f *addKnowledgeStoreRepository) FindByID(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true
	f.findByIDID = storeID

	if f.findByIDError != nil {
		return nil, f.findByIDError
	}

	return f.store, nil
}

func (f *addKnowledgeStoreRepository) FindByOwnerID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *addKnowledgeStoreRepository) FindBySlug(
	context.Context,
	string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *addKnowledgeStoreRepository) ExistsBySlug(
	context.Context,
	string,
) (bool, error) {
	return false, nil
}

func (f *addKnowledgeStoreRepository) ListActive(
	context.Context,
	string,
	int,
	int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *addKnowledgeStoreRepository) Update(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (f *addKnowledgeStoreRepository) ChangePlan(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (f *addKnowledgeStoreRepository) ChangeStatus(
	context.Context,
	uuid.UUID,
	string,
) error {
	return nil
}

type addKnowledgeDispatcher struct {
	called        bool
	knowledge     ports.StoreKnowledge
	dispatchError error
}

func (f *addKnowledgeDispatcher) DispatchStoreKnowledge(
	_ context.Context,
	knowledge ports.StoreKnowledge,
) error {
	f.called = true
	f.knowledge = knowledge

	return f.dispatchError
}

type addKnowledgeLogger struct {
	successCalled bool
}

func (f *addKnowledgeLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "store.add_knowledge.succeeded" {
		f.successCalled = true
	}

	return nil
}

type addKnowledgeMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *addKnowledgeMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *addKnowledgeMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
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
