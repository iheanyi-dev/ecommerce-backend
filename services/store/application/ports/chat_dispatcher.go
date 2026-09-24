package ports

import (
	"context"

	"github.com/google/uuid"
)

// ChatRequest contains the identity and conversation information required by
// the AI service to start a Store chat stream.
//
// UserID comes from the authenticated application context. The Store service
// forwards it to the AI service so the AI service can associate the chat with
// the registered user and manage conversation persistence.
type ChatRequest struct {
	StoreID        uuid.UUID
	UserID         uuid.UUID
	Message        string
	ConversationID *uuid.UUID
}

// ChatEvent represents one event produced by the AI chat stream.
//
// The Store application layer does not interpret or buffer the generated
// response. It forwards the stream returned by the AI dispatcher to the
// presentation layer.
type ChatEvent struct {
	Type            string
	Content         string
	ConversationID  *uuid.UUID
}

// ChatStream represents a live AI chat response stream.
//
// Recv retrieves the next event from the stream. The application use case
// deliberately does not call Recv; ownership of stream consumption belongs
// to the presentation/transport layer.
type ChatStream interface {
	Recv(ctx context.Context) (ChatEvent, error)
}

// ChatDispatcher starts a streaming chat session with the AI service.
//
// The dispatcher owns the integration with the AI service. Store only
// supplies Store/user/conversation context and the user's message.
type ChatDispatcher interface {
	StartChat(ctx context.Context, request ChatRequest) (ChatStream, error)
}
