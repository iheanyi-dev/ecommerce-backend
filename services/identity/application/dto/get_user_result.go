package dto

import "time"

// GetUserResult represents the safe application-level representation
// of a single user returned by an administrative user-management
// operation.
//
// Sensitive authentication information such as PasswordHash is
// deliberately excluded from this DTO.
type GetUserResult struct {
	ID        string
	FullName  string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
