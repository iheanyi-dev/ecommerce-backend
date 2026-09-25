package schemas

// UpdateStoreRequest represents the fields an authenticated Store owner
// may change.
type UpdateStoreRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}
