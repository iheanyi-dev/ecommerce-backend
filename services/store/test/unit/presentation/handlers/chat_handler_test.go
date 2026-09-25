package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type chatServiceMock struct {
	execute func(context.Context, dto.ChatInput) (ports.ChatStream, error)
}

func (m *chatServiceMock) Execute(
	ctx context.Context,
	input dto.ChatInput,
) (ports.ChatStream, error) {
	return m.execute(ctx, input)
}

type chatStreamMock struct {
	events []ports.ChatEvent
	index  int
	err    error
}

func (s *chatStreamMock) Recv(_ context.Context) (ports.ChatEvent, error) {
	if s.index < len(s.events) {
		event := s.events[s.index]
		s.index++
		return event, nil
	}

	if s.err != nil {
		return ports.ChatEvent{}, s.err
	}

	return ports.ChatEvent{}, io.EOF
}

func TestChatHandler(t *testing.T) {
	storeID := uuid.New()
	conversationID := uuid.New()

	t.Run("streams chat events as SSE", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, input dto.ChatInput) (ports.ChatStream, error) {
				require.Equal(t, storeID, input.StoreID)
				require.Equal(t, "hello", input.Message)
				require.NotNil(t, input.ConversationID)
				require.Equal(t, conversationID, *input.ConversationID)

				return &chatStreamMock{
					events: []ports.ChatEvent{
						{
							Type:           "token",
							Content:        "Hello",
							ConversationID: &conversationID,
						},
						{
							Type:           "token",
							Content:        " world",
							ConversationID: &conversationID,
						},
					},
				}, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{
				"message":"hello",
				"conversation_id":"`+conversationID.String()+`"
			}`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))

		assert.Equal(
			t,
			"data: {\"type\":\"token\",\"content\":\"Hello\",\"conversation_id\":\""+conversationID.String()+"\"}\n\n"+
				"data: {\"type\":\"token\",\"content\":\" world\",\"conversation_id\":\""+conversationID.String()+"\"}\n\n",
			recorder.Body.String(),
		)
	})

	t.Run("streams event without conversation id", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, input dto.ChatInput) (ports.ChatStream, error) {
				require.Nil(t, input.ConversationID)

				return &chatStreamMock{
					events: []ports.ChatEvent{
						{
							Type:    "complete",
							Content: "done",
						},
					},
				}, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{"message":"hello"}`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(
			t,
			recorder.Body.String(),
			`data: {"type":"complete","content":"done"}`,
		)
	})

	t.Run("returns application error", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				return nil, errors.New("chat failed")
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{"message":"hello"}`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("rejects non POST method", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				t.Fatal("service must not be called")
				return nil, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/"+storeID.String()+"/chat",
			nil,
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	})

	t.Run("rejects invalid store id", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				t.Fatal("service must not be called")
				return nil, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/not-a-uuid/chat",
			strings.NewReader(`{"message":"hello"}`),
		)
		request.SetPathValue("id", "not-a-uuid")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				t.Fatal("service must not be called")
				return nil, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{"message":`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("rejects invalid conversation id", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				t.Fatal("service must not be called")
				return nil, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{
				"message":"hello",
				"conversation_id":"invalid"
			}`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("stops cleanly when stream returns EOF", func(t *testing.T) {
		service := &chatServiceMock{
			execute: func(_ context.Context, _ dto.ChatInput) (ports.ChatStream, error) {
				return &chatStreamMock{
					events: []ports.ChatEvent{
						{
							Type:    "complete",
							Content: "done",
						},
					},
					err: io.EOF,
				}, nil
			},
		}

		handler := handlers.NewChatHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores/"+storeID.String()+"/chat",
			strings.NewReader(`{"message":"hello"}`),
		)
		request.SetPathValue("id", storeID.String())

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"content":"done"`)
	})
}
