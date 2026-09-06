package use_cases_test

import (
	"context"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// fakeUpdateProfileRepository is an in-memory repository used to test the
// profile update use case without requiring PostgreSQL.
type fakeUpdateProfileRepository struct {
	user *user.User

	updateFullNameCalled bool
	updatedUserID        user.UserID
	updatedFullName      user.FullName
}

// UpdateFullName records the requested profile update.
//
// The complete aggregate is supplied so the fake can verify that the
// domain-generated UpdatedAt value travels through the persistence boundary.
func (f *fakeUpdateProfileRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	f.updateFullNameCalled = true
	f.updatedUserID = existingUser.ID()
	f.updatedFullName = existingUser.FullName()

	return nil
}

// UpdatePasswordHash satisfies the UserRepository interface.
//
// These profile-update tests do not exercise password persistence,
// so the fake intentionally performs no operation.
func (f *fakeUpdateProfileRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// ExistsByEmail satisfies the UserRepository interface.
func (f *fakeUpdateProfileRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

// Create satisfies the UserRepository interface.
func (f *fakeUpdateProfileRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

// FindByEmail satisfies the UserRepository interface.
func (f *fakeUpdateProfileRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

// FindByID returns the test user.
func (f *fakeUpdateProfileRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	if f.user == nil {
		return nil, nil
	}

	return f.user, nil
}

// newProfileUpdateUser creates an active user for profile-update tests.
func newProfileUpdateUser(t *testing.T) *user.User {
	t.Helper()

	userID := user.NewUserID()

	fullName, err := user.NewFullName("Original Name")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("profile@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	now := time.Now().UTC()

	return user.ReconstituteUser(
		userID,
		fullName,
		email,
		passwordHash,
		user.RoleUser,
		user.StatusActive,
		now,
		now,
	)
}

// TestUpdateUserProfileUseCase_UpdateFullName verifies that an authenticated
// user can update their full name while immutable account fields remain
// unchanged.
func TestUpdateUserProfileUseCase_UpdateFullName(t *testing.T) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	useCase := use_cases.NewUpdateUserProfileUseCase(repository)

	command := dto.UpdateUserProfileCommand{
		FullName: "Updated Name",
	}

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		command,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.UserID != testUser.ID().String() {
		t.Fatalf("expected user ID %q, got %q",
			testUser.ID().String(),
			result.UserID,
		)
	}

	if result.FullName != "Updated Name" {
		t.Fatalf("expected full name %q, got %q",
			"Updated Name",
			result.FullName,
		)
	}

	// Email must remain unchanged because Phase 7 makes it immutable.
	if result.Email != "profile@example.com" {
		t.Fatalf("expected immutable email %q, got %q",
			"profile@example.com",
			result.Email,
		)
	}

	if !repository.updateFullNameCalled {
		t.Fatal("expected UpdateFullName to be called")
	}

	if repository.updatedUserID != testUser.ID() {
		t.Fatalf("expected updated user ID %q, got %q",
			testUser.ID(),
			repository.updatedUserID,
		)
	}

	if repository.updatedFullName.String() != "Updated Name" {
		t.Fatalf("expected updated full name %q, got %q",
			"Updated Name",
			repository.updatedFullName.String(),
		)
	}
}

// Compile-time assertion ensures the fake repository remains compatible
// with the application's repository contract.
var _ ports.UserRepository = (*fakeUpdateProfileRepository)(nil)
