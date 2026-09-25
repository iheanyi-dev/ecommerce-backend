package schemas

// AddStoreKnowledgeResponse confirms that a knowledge file was submitted
// for processing.
type AddStoreKnowledgeResponse struct {
	StoreID string `json:"store_id"`
}
