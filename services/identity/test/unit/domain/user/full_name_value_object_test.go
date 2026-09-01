package user_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestNewFullName_WithValidName_CreatesFullName(t *testing.T) {
	t.Parallel()

	name, err := user.NewFullName("John Doe")

	if err != nil {
		t.Fatalf("expected valid name to succeed, got error: %v", err)
	}

	if name.String() != "John Doe" {
		t.Fatalf("expected %q, got %q", "John Doe", name.String())
	}
}

func TestNewFullName_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	name, err := user.NewFullName("  John Doe  ")

	if err != nil {
		t.Fatalf("expected name creation to succeed, got error: %v", err)
	}

	if name.String() != "John Doe" {
		t.Fatalf("expected trimmed name, got %q", name.String())
	}
}

func TestNewFullName_WithInvalidName_ReturnsError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
	}{
		{
			name:  "empty",
			value: "",
		},
		{
			name:  "whitespace only",
			value: "   ",
		},
		{
			name:  "too long",
			value: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := user.NewFullName(testCase.value)

			if err == nil {
				t.Fatalf("expected %q to be rejected", testCase.value)
			}
		})
	}
}
