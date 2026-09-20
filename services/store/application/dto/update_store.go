package dto

import "github.com/google/uuid"

// UpdateStoreInput contains the fields an authenticated Store owner may
// change.
//
// Owner identity is deliberately not supplied by the caller. It is obtained
// from the authenticated application context.
type UpdateStoreInput struct {
	StoreID     uuid.UUID
	Name        string
	Slug        string
	Description string
}

// UpdateStoreOutput represents the updated Store.
type UpdateStoreOutput = StoreOutput
