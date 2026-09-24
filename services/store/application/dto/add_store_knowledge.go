package dto

import "github.com/google/uuid"

// StoreKnowledgeFileInput represents a knowledge file supplied to the AI
// service for processing.
type StoreKnowledgeFileInput struct {
	// Content contains the raw file bytes.
	Content []byte

	// Filename identifies the supplied file.
	Filename string

	// ContentType identifies the file media type.
	ContentType string
}

// AddStoreKnowledgeInput contains the data required to add knowledge to a
// Store's AI knowledge base.
//
// The Store ID identifies the target Store. No user ID is required because
// the knowledge operation is scoped to the Store.
type AddStoreKnowledgeInput struct {
	StoreID uuid.UUID
	File    StoreKnowledgeFileInput
}

// AddStoreKnowledgeOutput represents the result of successfully submitting
// the knowledge file for processing.
type AddStoreKnowledgeOutput struct {
	StoreID uuid.UUID
}
