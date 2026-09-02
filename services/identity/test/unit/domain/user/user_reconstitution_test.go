package user_test

import (
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// TestReconstituteUserPreservesPersistedState verifies that reconstituting a
// User aggregate preserves every piece of persisted state exactly.
//
// This test is important because ReconstituteUser must behave differently
// from NewUser. NewUser creates fresh state, while ReconstituteUser restores
// existing state.
func TestReconstituteUserPreservesPersistedState(t *testing.T) {
	// Create the domain values that represent the persisted user.
	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("john@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("$2a$10$example-password-hash")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	// Use an explicitly generated UserID to represent the ID that came from
	// persistent storage.
	persistedID := user.NewUserID()

	// These values represent the state that would have been loaded from
	// PostgreSQL.
	persistedRole := user.RoleVendor
	persistedStatus := user.StatusActive
	persistedCreatedAt := time.Date(
		2026,
		time.January,
		15,
		10,
		30,
		0,
		0,
		time.UTC,
	)
	persistedUpdatedAt := time.Date(
		2026,
		time.February,
		20,
		14,
		45,
		0,
		0,
		time.UTC,
	)

	// Reconstitute the aggregate from the persisted values.
	reconstitutedUser := user.ReconstituteUser(
		persistedID,
		fullName,
		email,
		passwordHash,
		persistedRole,
		persistedStatus,
		persistedCreatedAt,
		persistedUpdatedAt,
	)

	// Verify that the aggregate retained the persisted identity.
	if reconstitutedUser.ID() != persistedID {
		t.Fatalf(
			"expected user ID %s, got %s",
			persistedID.String(),
			reconstitutedUser.ID().String(),
		)
	}

	// Verify that the persisted descriptive information was preserved.
	if reconstitutedUser.FullName() != fullName {
		t.Fatalf(
			"expected full name %q, got %q",
			fullName.String(),
			reconstitutedUser.FullName().String(),
		)
	}

	if reconstitutedUser.Email() != email {
		t.Fatalf(
			"expected email %q, got %q",
			email.String(),
			reconstitutedUser.Email().String(),
		)
	}

	// The password hash must be restored exactly as persisted. The domain
	// never exposes the plaintext password.
	if reconstitutedUser.PasswordHash() != passwordHash {
		t.Fatalf("reconstituted password hash does not match persisted hash")
	}

	// Verify that authentication-relevant account state was preserved.
	if reconstitutedUser.Role() != persistedRole {
		t.Fatalf(
			"expected role %q, got %q",
			persistedRole.String(),
			reconstitutedUser.Role().String(),
		)
	}

	if reconstitutedUser.Status() != persistedStatus {
		t.Fatalf(
			"expected status %q, got %q",
			persistedStatus.String(),
			reconstitutedUser.Status().String(),
		)
	}

	// Timestamps are persisted state as well and must not be replaced with
	// time.Now() during reconstitution.
	if !reconstitutedUser.CreatedAt().Equal(persistedCreatedAt) {
		t.Fatalf(
			"expected created_at %v, got %v",
			persistedCreatedAt,
			reconstitutedUser.CreatedAt(),
		)
	}

	if !reconstitutedUser.UpdatedAt().Equal(persistedUpdatedAt) {
		t.Fatalf(
			"expected updated_at %v, got %v",
			persistedUpdatedAt,
			reconstitutedUser.UpdatedAt(),
		)
	}
}