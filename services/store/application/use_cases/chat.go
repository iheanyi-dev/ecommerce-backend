package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// ChatUseCase starts a streaming AI chat for a Store.
//
// Store is responsible for authorization and AI capability checks. The AI
// service owns conversation persistence, knowledge retrieval, inference, and
// generation. Consequently, this use case starts the stream and returns it
// without consuming or buffering any generated content.
type ChatUseCase struct {
	storeRepository ports.StoreRepository
	chatDispatcher  ports.ChatDispatcher
	logger          ports.Logger
	metrics         ports.Metrics
}

// NewChatUseCase creates a Chat use case with its required application ports.
func NewChatUseCase(
	storeRepository ports.StoreRepository,
	chatDispatcher ports.ChatDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChatUseCase {
	return &ChatUseCase{
		storeRepository: storeRepository,
		chatDispatcher:  chatDispatcher,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute authenticates and authorizes the chat request, then starts the AI
// response stream.
//
// The returned ChatStream is the exact stream supplied by the dispatcher.
// Execute does not call Recv and therefore does not buffer the AI response.
func (uc *ChatUseCase) Execute(
	ctx context.Context,
	input dto.ChatInput,
) (ports.ChatStream, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_chat_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "chat",
		},
	})

	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)
	if !authenticated {
		return uc.fail(
			ctx,
			startedAt,
			"authentication",
			applicationerrors.ErrUnauthenticated,
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"repository_lookup",
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
		return uc.fail(
			ctx,
			startedAt,
			"authorization",
			applicationerrors.ErrStoreNotFound,
		)
	}

	if !store.Plan().AIFeaturesAllowed() {
		return uc.fail(
			ctx,
			startedAt,
			"ai_not_allowed",
			fmt.Errorf("store plan does not allow AI features"),
		)
	}

	stream, err := uc.chatDispatcher.StartChat(
		ctx,
		ports.ChatRequest{
			StoreID:        store.ID().Value(),
			UserID:         identity.UserID,
			Message:        input.Message,
			ConversationID: input.ConversationID,
		},
	)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"chat_dispatch",
			fmt.Errorf("start chat: %w", err),
		)
	}

	if err := uc.log(ctx, ports.LogEvent{
		Event:     "store.chat.succeeded",
		Operation: "chat",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	}); err != nil {
		// Observability failure must not terminate a successfully started
		// streaming chat.
		_ = err
	}

	uc.increment(ctx, ports.Metric{
		Name:  "store_chat_success_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "chat",
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_chat_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "chat",
			"result":    "success",
		},
	})

	return stream, nil
}

func (uc *ChatUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (ports.ChatStream, error) {
	identity, authenticated := ports.AuthenticatedIdentityFromContext(ctx)

	userID := ""
	role := ""

	if authenticated {
		userID = identity.UserID.String()
		role = identity.Role
	}

	if logErr := uc.log(ctx, ports.LogEvent{
		Event:           "store.chat.failed",
		Operation:       "chat",
		UserID:          userID,
		Role:            role,
		FailureCategory: failureCategory,
	}); logErr != nil {
		_ = logErr
	}

	uc.increment(ctx, ports.Metric{
		Name:  "store_chat_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "chat",
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_chat_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "chat",
			"result":    "failure",
		},
	})

	return nil, err
}

func (uc *ChatUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	if uc.logger == nil {
		return nil
	}

	return uc.logger.Log(ctx, event)
}

func (uc *ChatUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *ChatUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, metric)
}
