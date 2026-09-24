package dto

import "github.com/google/uuid"

// ChatInput contains the information required to start a streaming chat
// interaction for a Store.
//
// User identity is intentionally not part of this input. The authenticated
// user is obtained from the application authentication context so callers
// cannot supply or impersonate another user.
type ChatInput struct {
	StoreID        uuid.UUID
	Message        string
	ConversationID *uuid.UUID
}
