package user

import (
	"errors"
	"strings"
)

var ErrInvalidStatus = errors.New("invalid user status")

// Status represents the lifecycle state of a user account.
type Status string

const (
	StatusPendingVerification Status = "pending_verification"
	StatusActive              Status = "active"
	StatusSuspended           Status = "suspended"
	StatusInactive            Status = "inactive"
)

// NewStatus creates a validated user status.
func NewStatus(value string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(value)))

	switch status {
	case StatusPendingVerification,
		StatusActive,
		StatusSuspended,
		StatusInactive:
		return status, nil
	default:
		return "", ErrInvalidStatus
	}
}

// String returns the status's serialized representation.
func (status Status) String() string {
	return string(status)
}
