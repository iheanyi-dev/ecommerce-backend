package user_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestUserStatus_ContainsSupportedStatuses(t *testing.T) {
	t.Parallel()

	statuses := []user.Status{
		user.StatusPendingVerification,
		user.StatusActive,
		user.StatusSuspended,
		user.StatusInactive,
	}

	for _, status := range statuses {
		if status.String() == "" {
			t.Fatal("expected status to have a value")
		}
	}
}

func TestNewStatus_WithInvalidValue_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := user.NewStatus("deleted")

	if err == nil {
		t.Fatal("expected invalid status to be rejected")
	}
}

func TestNewStatus_WithValidValue_CreatesStatus(t *testing.T) {
	t.Parallel()

	status, err := user.NewStatus("active")

	if err != nil {
		t.Fatalf("expected status creation to succeed, got error: %v", err)
	}

	if status != user.StatusActive {
		t.Fatalf("expected active status, got %v", status)
	}
}
