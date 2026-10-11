package events

import "github.com/google/uuid"

const (
	KnowledgeIngestionRequestedTopic = "knowledge.ingestion.requested"
	KnowledgeDeletionRequestedTopic  = "knowledge.deletion.requested"
)

type KnowledgeIngestionRequested struct {
	KnowledgeID      uuid.UUID `json:"knowledge_id"`
	StoreID          uuid.UUID `json:"store_id"`
	StorageReference string    `json:"storage_reference"`
}

type KnowledgeDeletionRequested struct {
	KnowledgeID uuid.UUID `json:"knowledge_id"`
	StoreID     uuid.UUID `json:"store_id"`
}
