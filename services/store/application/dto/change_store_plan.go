package dto

import "github.com/google/uuid"

// ChangeStorePlanInput contains the information required to request a Store
// plan change.
//
// The authenticated owner is deliberately not included here. Ownership is
// derived from the trusted authenticated identity in the application context,
// rather than from caller-supplied data.
//
// PlanType remains a string at the DTO boundary so the application DTO does
// not depend on the Store domain's Plan value object.
type ChangeStorePlanInput struct {
	StoreID  uuid.UUID
	PlanType string
}
