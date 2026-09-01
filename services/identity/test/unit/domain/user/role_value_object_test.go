package user_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestUserRole_ContainsSupportedRoles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role user.Role
	}{
		{
			name: "admin",
			role: user.RoleAdmin,
		},
		{
			name: "vendor",
			role: user.RoleVendor,
		},
		{
			name: "user",
			role: user.RoleUser,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.role.String() == "" {
				t.Fatal("expected role to have a value")
			}
		})
	}
}

func TestNewRole_WithInvalidValue_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := user.NewRole("superuser")

	if err == nil {
		t.Fatal("expected invalid role to be rejected")
	}
}

func TestNewRole_WithValidValue_CreatesRole(t *testing.T) {
	t.Parallel()

	role, err := user.NewRole("vendor")

	if err != nil {
		t.Fatalf("expected role creation to succeed, got error: %v", err)
	}

	if role != user.RoleVendor {
		t.Fatalf("expected vendor role, got %v", role)
	}
}
