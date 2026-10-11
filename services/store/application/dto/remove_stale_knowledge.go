package dto

import "github.com/google/uuid"

type RemoveStaleKnowledgeInput struct {
	StoreID     uuid.UUID
	KnowledgeID uuid.UUID
}

type RemoveStaleKnowledgeOutput struct {
	StoreID     uuid.UUID
	KnowledgeID uuid.UUID
}
