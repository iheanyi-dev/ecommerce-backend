package dto

// UpdateUserProfileCommand contains the mutable profile fields a user may
// change through the self-service account endpoint.
//
// Email is intentionally excluded because the user's email is immutable
// in the Phase 7 account-management workflow.
type UpdateUserProfileCommand struct {
	FullName string
}
