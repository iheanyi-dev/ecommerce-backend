package dto

import "github.com/google/uuid"

// ChangeStoreStatusInput contains the Store status requested by an authorized
// internal caller such as the Billing/Subscription service.
//
// Authorization and subscription validation are deliberately outside the
// application use case. The presentation/service boundary is responsible for
// ensuring this request is trusted before it reaches the application layer.
type ChangeStoreStatusInput struct {
	StoreID uuid.UUID
	Status  string
}
