package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// ChatHandler exposes the Store AI chat stream over HTTP.
//
// The application layer owns authentication, Store ownership, and AI-plan
// authorization. The handler is responsible only for HTTP decoding and
// translating the returned ChatStream into an SSE response.
type ChatHandler struct {
	service ports.ChatService
}

// NewChatHandler creates a Chat HTTP handler.
func NewChatHandler(service ports.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

// ServeHTTP starts and streams a Store AI conversation.
//
// SSE is used because ChatStream is inherently incremental: buffering the
// complete response would defeat the purpose of the application stream.
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presentationerrors.WriteError(w, presentationerrors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentationerrors.WriteError(w, err)
		return
	}

	var request schemas.ChatRequest
	if err := decodeJSON(r, &request); err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}

	var conversationID *uuid.UUID
	if request.ConversationID != nil {
		parsed, err := uuid.Parse(*request.ConversationID)
		if err != nil {
			presentationerrors.WriteError(w, err)
			return
		}
		conversationID = &parsed
	}

	stream, err := h.service.Execute(
		r.Context(),
		dto.ChatInput{
			StoreID:        storeID,
			Message:        request.Message,
			ConversationID: conversationID,
		},
	)
	if err != nil {
		presentationerrors.WriteError(w, err)
		return
	}

	h.stream(w, r, stream)
}

// stream consumes the application stream and writes each event as an SSE
// message. Once the HTTP headers have been committed, stream errors cannot
// be translated into the normal JSON error response, so the connection is
// terminated instead.
func (h *ChatHandler) stream(
	w http.ResponseWriter,
	r *http.Request,
	stream ports.ChatStream,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	for {
		event, err := stream.Recv(r.Context())
		if errors.Is(err, io.EOF) {
			return
		}

		if err != nil {
			return
		}

		response := schemas.ChatEventResponse{
			Type:           event.Type,
			Content:        event.Content,
			ConversationID: uuidPointerToStringPointer(event.ConversationID),
		}

		payload, err := json.Marshal(response)
		if err != nil {
			return
		}

		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return
		}

		flusher.Flush()
	}
}

func uuidPointerToStringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	result := value.String()
	return &result
}
