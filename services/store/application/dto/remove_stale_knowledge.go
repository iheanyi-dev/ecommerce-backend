package dto

import "github.com/google/uuid"

// RemoveStaleKnowledgeInput identifies the Store knowledge file that should
// be removed.
//
// Store authorizes the request, while the AI service owns the actual
// knowledge file and its persistence.
type RemoveStaleKnowledgeInput struct {
	StoreID  uuid.UUID
	Filename string
}

// RemoveStaleKnowledgeOutput confirms the Store and knowledge file for which
// removal was requested.
type RemoveStaleKnowledgeOutput struct {
	StoreID  uuid.UUID
	Filename string
}
