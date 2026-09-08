package dto

// UpdateUserStatusCommand contains the account-status change requested by an
// administrator.
//
// The status is intentionally the only field in this command. Administrative
// status management must never modify profile fields, password, or role.
type UpdateUserStatusCommand struct {
	Status string
}
