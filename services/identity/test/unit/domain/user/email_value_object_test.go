package user_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestNewEmail_WithValidEmail_CreatesEmail(t *testing.T) {
	t.Parallel()

	email, err := user.NewEmail("John@example.com")

	if err != nil {
		t.Fatalf("expected email creation to succeed, got error: %v", err)
	}

	if email.String() != "john@example.com" {
		t.Fatalf(
			"expected normalized email %q, got %q",
			"john@example.com",
			email.String(),
		)
	}
}

func TestNewEmail_WithInvalidEmail_ReturnsError(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"",
		"john",
		"john@",
		"@example.com",
		"john example@example.com",
	}

	for _, value := range testCases {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			_, err := user.NewEmail(value)

			if err == nil {
				t.Fatalf("expected email %q to be rejected", value)
			}
		})
	}
}
