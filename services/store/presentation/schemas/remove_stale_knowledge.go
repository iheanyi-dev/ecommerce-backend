package schemas

// RemoveStaleKnowledgeRequest identifies the knowledge file to remove.
type RemoveStaleKnowledgeRequest struct {
	KnowledgeID string `json:"knowledge_id"`
}

// RemoveStaleKnowledgeResponse confirms the removal request.
type RemoveStaleKnowledgeResponse struct {
	StoreID     string `json:"store_id"`
	KnowledgeID string `json:"knowledge_id"`
}
