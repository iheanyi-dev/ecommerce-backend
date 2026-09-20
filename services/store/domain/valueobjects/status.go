package valueobjects

import domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"

// Status represents the lifecycle state of a Store.
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// NewStatus creates a valid Store status.
func NewStatus(status Status) (Status, error) {
	switch status {
	case StatusActive, StatusInactive:
		return status, nil
	default:
		return "", domainerrors.ErrInvalidStatus
	}
}

// CanTransitionTo determines whether a Store can transition from the
// current status to the supplied target status.
func (s Status) CanTransitionTo(target Status) bool {
	switch s {
	case StatusActive:
		return target == StatusActive || target == StatusInactive
	case StatusInactive:
		return target == StatusInactive || target == StatusActive
	default:
		return false
	}
}
