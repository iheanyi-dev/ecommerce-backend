package user

import (
	"strings"

	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
)

// Role represents the account category within the platform.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleVendor Role = "vendor"
	RoleUser   Role = "user"
)

// NewRole creates a validated user role.
func NewRole(value string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(value)))

	switch role {
	case RoleAdmin, RoleVendor, RoleUser:
		return role, nil
	default:
		return "", domain_errors.ErrInvalidRole
	}
}

// String returns the role's serialized representation.
func (role Role) String() string {
	return string(role)
}
