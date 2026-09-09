package errors_test

import (
	"errors"
	"testing"

	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
)

// TestDomainErrorsAreDefined verifies that the domain layer exposes
// stable sentinel errors for domain-rule violations.
func TestDomainErrorsAreDefined(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "invalid email",
			err:  domain_errors.ErrInvalidEmail,
		},
		{
			name: "invalid full name",
			err:  domain_errors.ErrInvalidFullName,
		},
		{
			name: "invalid password hash",
			err:  domain_errors.ErrInvalidPasswordHash,
		},
		{
			name: "invalid role",
			err:  domain_errors.ErrInvalidRole,
		},
		{
			name: "invalid status",
			err:  domain_errors.ErrInvalidStatus,
		},
		{
			name: "invalid status transition",
			err:  domain_errors.ErrInvalidStatusTransition,
		},
		{
			name: "invalid user ID",
			err:  domain_errors.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf(
					"expected %s error to be defined",
					tt.name,
				)
			}

			if !errors.Is(tt.err, tt.err) {
				t.Fatalf(
					"expected %s to be identifiable with errors.Is",
					tt.name,
				)
			}
		})
	}
}
