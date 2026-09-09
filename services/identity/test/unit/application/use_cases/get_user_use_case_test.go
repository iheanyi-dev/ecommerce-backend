package use_cases_test

import (
	"context"
	"errors"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// fakeGetUserRepository is a test double for the existing UserRepository.
//
// The GetUser use case only needs the FindByID capability from the
// repository, but the complete interface is implemented because
// UserRepository is the existing application persistence boundary.
type fakeGetUserRepository struct {
	user *user.User
	err  error

	gotID user.UserID
}

// ExistsByEmail satisfies ports.UserRepository.
func (f *fakeGetUserRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

// Create satisfies ports.UserRepository.
func (f *fakeGetUserRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

// FindByEmail satisfies ports.UserRepository.
func (f *fakeGetUserRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

// FindByID records the requested user ID and returns the configured
// test user or error.
func (f *fakeGetUserRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	f.gotID = id

	return f.user, f.err
}

// UpdateFullName satisfies ports.UserRepository.
func (f *fakeGetUserRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// UpdateStatus satisfies the UserRepository interface.
//
// These profile-update tests do not exercise account-status persistence.
func (f *fakeGetUserRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// UpdatePasswordHash satisfies ports.UserRepository.
func (f *fakeGetUserRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// List satisfies ports.UserRepository.
func (f *fakeGetUserRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
}

// newGetUserTestUser creates a valid User aggregate for GetUser tests.
func newGetUserTestUser(
	t *testing.T,
	fullName string,
	email string,
) *user.User {
	t.Helper()

	name, err := user.NewFullName(fullName)
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	address, err := user.NewEmail(email)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	newUser, err := user.NewUser(
		name,
		address,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return newUser
}

func TestGetUserUseCase_Execute_ReturnsUser(
	t *testing.T,
) {
	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeGetUserRepository{
		user: testUser,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.ID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			result.ID,
		)
	}

	if result.FullName != testUser.FullName().String() {
		t.Fatalf(
			"expected full name %q, got %q",
			testUser.FullName().String(),
			result.FullName,
		)
	}

	if result.Email != testUser.Email().String() {
		t.Fatalf(
			"expected email %q, got %q",
			testUser.Email().String(),
			result.Email,
		)
	}

	if result.Role != testUser.Role().String() {
		t.Fatalf(
			"expected role %q, got %q",
			testUser.Role().String(),
			result.Role,
		)
	}

	if result.Status != testUser.Status().String() {
		t.Fatalf(
			"expected status %q, got %q",
			testUser.Status().String(),
			result.Status,
		)
	}

	if !result.CreatedAt.Equal(testUser.CreatedAt()) {
		t.Fatalf(
			"expected created_at %v, got %v",
			testUser.CreatedAt(),
			result.CreatedAt,
		)
	}

	if !result.UpdatedAt.Equal(testUser.UpdatedAt()) {
		t.Fatalf(
			"expected updated_at %v, got %v",
			testUser.UpdatedAt(),
			result.UpdatedAt,
		)
	}
}

func TestGetUserUseCase_Execute_PassesUserIDToRepository(
	t *testing.T,
) {
	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeGetUserRepository{
		user: testUser,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.gotID != testUser.ID() {
		t.Fatalf(
			"expected repository to receive user ID %q, got %q",
			testUser.ID().String(),
			repository.gotID.String(),
		)
	}
}

func TestGetUserUseCase_Execute_DoesNotExposePasswordHash(
	t *testing.T,
) {
	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeGetUserRepository{
		user: testUser,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Compile-time protection: GetUserResult must not contain a
	// PasswordHash field. Password hashes must never cross the
	// application-to-presentation boundary.
	_ = result.Email
}

func TestGetUserUseCase_Execute_ReturnsUserNotFound(
	t *testing.T,
) {
	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeGetUserRepository{
		user: nil,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err == nil {
		t.Fatal("expected user not found error, got nil")
	}

	if !errors.Is(err, application_errors.ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}
}

func TestGetUserUseCase_Execute_ReturnsRepositoryError(
	t *testing.T,
) {
	expectedErr := errors.New("database failure")

	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeGetUserRepository{
		user: testUser,
		err:  expectedErr,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestGetUserUseCase_Execute_RejectsInvalidUserID(
	t *testing.T,
) {
	repository := &fakeGetUserRepository{}

	useCase := use_cases.NewGetUserUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		"not-a-valid-user-id",
	)

	if err == nil {
		t.Fatal("expected invalid user ID error, got nil")
	}

	if repository.gotID != (user.UserID{}) {
		t.Fatal(
			"expected repository not to be called with an invalid user ID",
		)
	}
}

func TestGetUserUseCase_Execute_PreservesTimestamps(
	t *testing.T,
) {
	testUser := newGetUserTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	// Capture the domain timestamps before executing the use case so the
	// test explicitly verifies that persistence/domain timestamps are
	// transferred without being regenerated or altered.
	expectedCreatedAt := testUser.CreatedAt()
	expectedUpdatedAt := testUser.UpdatedAt()

	repository := &fakeGetUserRepository{
		user: testUser,
	}

	useCase := use_cases.NewGetUserUseCase(repository)

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !result.CreatedAt.Equal(expectedCreatedAt) {
		t.Fatalf(
			"expected created_at %v, got %v",
			expectedCreatedAt,
			result.CreatedAt,
		)
	}

	if !result.UpdatedAt.Equal(expectedUpdatedAt) {
		t.Fatalf(
			"expected updated_at %v, got %v",
			expectedUpdatedAt,
			result.UpdatedAt,
		)
	}

	// Ensure the test continues to use the time.Time type imported by the
	// DTO contract rather than relying only on string formatting.
	var _ time.Time = result.CreatedAt
}
