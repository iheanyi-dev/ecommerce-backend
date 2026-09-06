package use_cases_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
)

// TestErrUserNotFound verifies that the application exposes a stable
// sentinel error for missing user accounts.
func TestErrUserNotFound(t *testing.T) {
	if use_cases.ErrUserNotFound == nil {
		t.Fatal("expected ErrUserNotFound to be defined")
	}

	if !errors.Is(use_cases.ErrUserNotFound, use_cases.ErrUserNotFound) {
		t.Fatal("expected ErrUserNotFound to be identifiable with errors.Is")
	}
}
