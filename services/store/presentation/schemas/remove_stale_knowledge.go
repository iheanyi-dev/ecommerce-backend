package schemas

// RemoveStaleKnowledgeRequest identifies the knowledge file to remove.
type RemoveStaleKnowledgeRequest struct {
	Filename string `json:"filename"`
}

// RemoveStaleKnowledgeResponse confirms the removal request.
type RemoveStaleKnowledgeResponse struct {
	StoreID  string `json:"store_id"`
	Filename string `json:"filename"`
}
