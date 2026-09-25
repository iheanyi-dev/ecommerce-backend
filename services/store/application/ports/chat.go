package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ChatService starts a streaming AI chat interaction for an authorized
// Store request.
//
// The returned ChatStream is consumed by the presentation layer so the
// HTTP transport can forward events to the client as they arrive.
type ChatService interface {
	Execute(ctx context.Context, input dto.ChatInput) (ChatStream, error)
}
