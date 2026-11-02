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

func TestChatUseCase_RequiresAuthentication(t *testing.T) {
	repository := &chatStoreRepository{}
	dispatcher := &chatDispatcher{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChatInput{
			StoreID: uuid.New(),
			Message: "What products do you have?",
		},
	)

	if !errors.Is(err, applicationerrors.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}

	if repository.findByIDCalled {
		t.Fatal("expected repository not to be called")
	}

	if dispatcher.called {
		t.Fatal("expected chat dispatcher not to be called")
	}
}

func TestChatUseCase_StoreNotFound(t *testing.T) {
	repository := &chatStoreRepository{}
	dispatcher := &chatDispatcher{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	userID := uuid.New()
	storeID := uuid.New()

	ctx := chatAuthenticatedContext(context.Background(), userID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: storeID,
			Message: "What products do you have?",
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if repository.findByIDID != storeID {
		t.Fatalf(
			"expected repository lookup for %s, got %s",
			storeID,
			repository.findByIDID,
		)
	}

	if dispatcher.called {
		t.Fatal("expected chat dispatcher not to be called")
	}
}

func TestChatUseCase_RepositoryLookupFailure(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	repository := &chatStoreRepository{
		findByIDError: expectedErr,
	}
	dispatcher := &chatDispatcher{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), uuid.New())

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: uuid.New(),
			Message: "What products do you have?",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error to be wrapped, got %v", err)
	}

	if dispatcher.called {
		t.Fatal("expected chat dispatcher not to be called")
	}
}

func TestChatUseCase_NonOwnerReceivesStoreNotFound(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &chatStoreRepository{
		store: store,
	}
	dispatcher := &chatDispatcher{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), otherUserID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: store.ID().Value(),
			Message: "What products do you have?",
		},
	)

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf("expected ErrStoreNotFound, got %v", err)
	}

	if dispatcher.called {
		t.Fatal("expected chat dispatcher not to be called")
	}
}

func TestChatUseCase_RejectsStoreWithoutAIFeatures(t *testing.T) {
	ownerID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
	)

	repository := &chatStoreRepository{
		store: store,
	}
	dispatcher := &chatDispatcher{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: store.ID().Value(),
			Message: "What products do you have?",
		},
	)

	if err == nil {
		t.Fatal("expected AI capability error")
	}

	if dispatcher.called {
		t.Fatal("expected chat dispatcher not to be called")
	}
}

func TestChatUseCase_StartsStreamingChatAndReturnsStreamUnchanged(t *testing.T) {
	ownerID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &chatStoreRepository{
		store: store,
	}

	stream := &chatStream{}
	dispatcher := &chatDispatcher{
		stream: stream,
	}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), ownerID)

	input := dto.ChatInput{
		StoreID: store.ID().Value(),
		Message: "Which products are available?",
	}

	result, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected a streaming result")
	}

	if result != stream {
		t.Fatal("expected use case to return the exact stream supplied by the dispatcher")
	}

	if !dispatcher.called {
		t.Fatal("expected chat dispatcher to be called")
	}

	if dispatcher.request.StoreID != store.ID().Value() {
		t.Fatalf(
			"expected dispatched StoreID %s, got %s",
			store.ID().Value(),
			dispatcher.request.StoreID,
		)
	}

	if dispatcher.request.UserID != ownerID {
		t.Fatalf(
			"expected dispatched UserID %s, got %s",
			ownerID,
			dispatcher.request.UserID,
		)
	}

	if dispatcher.request.Message != input.Message {
		t.Fatalf(
			"expected dispatched message %q, got %q",
			input.Message,
			dispatcher.request.Message,
		)
	}

	if stream.recvCalled {
		t.Fatal("expected use case not to consume the streaming response")
	}
}

func TestChatUseCase_ForwardsConversationID(t *testing.T) {
	ownerID := uuid.New()
	conversationID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &chatStoreRepository{
		store: store,
	}

	dispatcher := &chatDispatcher{
		stream: &chatStream{},
	}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID:        store.ID().Value(),
			Message:        "Continue our conversation.",
			ConversationID: &conversationID,
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dispatcher.request.ConversationID == nil {
		t.Fatal("expected conversation ID to be forwarded")
	}

	if *dispatcher.request.ConversationID != conversationID {
		t.Fatalf(
			"expected conversation ID %s, got %s",
			conversationID,
			*dispatcher.request.ConversationID,
		)
	}
}

func TestChatUseCase_DispatchFailureIsReturned(t *testing.T) {
	expectedErr := errors.New("AI service unavailable")

	ownerID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &chatStoreRepository{
		store: store,
	}

	dispatcher := &chatDispatcher{
		dispatchError: expectedErr,
	}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		&chatLogger{},
		&chatMetrics{},
	)

	ctx := chatAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: store.ID().Value(),
			Message: "What products do you have?",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected dispatcher error to be returned, got %v", err)
	}
}

func TestChatUseCase_ObservesSuccessfulOperation(t *testing.T) {
	ownerID := uuid.New()

	store := newChatStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
	)

	repository := &chatStoreRepository{
		store: store,
	}

	dispatcher := &chatDispatcher{
		stream: &chatStream{},
	}

	logger := &chatLogger{}
	metrics := &chatMetrics{}

	uc := usecases.NewChatUseCase(
		repository,
		dispatcher,
		logger,
		metrics,
	)

	ctx := chatAuthenticatedContext(context.Background(), ownerID)

	_, err := uc.Execute(
		ctx,
		dto.ChatInput{
			StoreID: store.ID().Value(),
			Message: "What products do you have?",
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

type chatStoreRepository struct {
	store         *entities.Store
	findByIDError error
	findByIDCalled bool
	findByIDID    uuid.UUID
}

func (f *chatStoreRepository) Create(context.Context, *entities.Store) error {
	return nil
}

func (f *chatStoreRepository) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (f *chatStoreRepository) FindByID(
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

func (f *chatStoreRepository) FindByOwnerID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *chatStoreRepository) FindBySlug(
	context.Context,
	string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *chatStoreRepository) ExistsBySlug(
	context.Context,
	string,
) (bool, error) {
	return false, nil
}

func (f *chatStoreRepository) ListActive(
	context.Context,
	string,
	int,
	int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *chatStoreRepository) Update(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (f *chatStoreRepository) ChangePlan(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (f *chatStoreRepository) ChangeStatus(
	context.Context,
	uuid.UUID,
	string,
) error {
	return nil
}

type chatDispatcher struct {
	called        bool
	request       ports.ChatRequest
	stream        ports.ChatStream
	dispatchError error
}

func (f *chatDispatcher) StartChat(
	_ context.Context,
	request ports.ChatRequest,
) (ports.ChatStream, error) {
	f.called = true
	f.request = request

	if f.dispatchError != nil {
		return nil, f.dispatchError
	}

	return f.stream, nil
}

type chatStream struct {
	recvCalled bool
}

func (f *chatStream) Recv(context.Context) (ports.ChatEvent, error) {
	f.recvCalled = true
	return ports.ChatEvent{}, nil
}

type chatLogger struct {
	successCalled bool
}

func (f *chatLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "store.chat.succeeded" {
		f.successCalled = true
	}

	return nil
}

type chatMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *chatMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *chatMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return nil
}

func newChatStore(
	t *testing.T,
	ownerID uuid.UUID,
	planType valueobjects.PlanType,
) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		ownerID,
		"Chat Store",
		"chat-store",
		"Chat test store",
		nil,
		string(planType),
	)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	return store
}

func chatAuthenticatedContext(
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

// Keep time imported as part of this test's explicit dependency set while
// the observability fakes remain intentionally minimal.
var _ = time.Second
