package dto

import "github.com/google/uuid"

// ChangeStoreImageInput contains the data required to replace a Store image.
//
// The Store ID identifies the Store being updated. The authenticated owner
// identity is deliberately not included here; it is obtained from the
// application authentication context.
type ChangeStoreImageInput struct {
	StoreID uuid.UUID
	Image   *StoreImageInput
}

// ChangeStoreImageOutput represents the Store after its image reference
// has been successfully changed and persisted.
type ChangeStoreImageOutput = StoreOutput
