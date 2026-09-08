package dto

import "time"

// ListUsersResult represents the safe application-level representation
// of users returned by an administrative user-listing operation.
//
// Sensitive authentication data such as PasswordHash is deliberately
// excluded from this DTO.
type ListUsersResult struct {
	Users []UserSummary
}

// UserSummary contains the non-sensitive information that may be exposed
// when displaying a user's account to an authorized administrator.
type UserSummary struct {
	ID        string
	FullName  string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
