package schemas

// ChatRequest contains the message sent to the Store AI chat.
//
// ConversationID is optional because the AI service may create the
// conversation on the first message.
type ChatRequest struct {
	Message        string  `json:"message"`
	ConversationID *string `json:"conversation_id,omitempty"`
}

// ChatEventResponse is the transport representation of one streaming
// chat event.
type ChatEventResponse struct {
	Type           string  `json:"type"`
	Content        string  `json:"content"`
	ConversationID *string `json:"conversation_id,omitempty"`
}
