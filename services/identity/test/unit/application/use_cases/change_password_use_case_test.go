package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// -----------------------------------------------------------------------------
// Test doubles
// -----------------------------------------------------------------------------

// fakeChangePasswordRepository is an in-memory implementation of
// ports.UserRepository.
//
// The fake deliberately records calls so the tests can verify the exact
// application workflow without involving PostgreSQL.
type fakeChangePasswordRepository struct {
	user *user.User

	findByIDCalled bool
	findByIDError  error

	updatePasswordHashCalled bool
	updatePasswordHashError  error

	updatedUserID       user.UserID
	updatedPasswordHash user.PasswordHash
}

func (f *fakeChangePasswordRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakeChangePasswordRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

func (f *fakeChangePasswordRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeChangePasswordRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	f.findByIDCalled = true

	if f.findByIDError != nil {
		return nil, f.findByIDError
	}

	if f.user == nil {
		return nil, nil
	}

	if f.user.ID() != id {
		return nil, nil
	}

	return f.user, nil
}

func (f *fakeChangePasswordRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeChangePasswordRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	f.updatePasswordHashCalled = true
	f.updatedUserID = existingUser.ID()
	f.updatedPasswordHash = existingUser.PasswordHash()

	return f.updatePasswordHashError
}

// fakeChangePasswordHasher allows the tests to verify the password hashing
// workflow independently from bcrypt or another concrete algorithm.
type fakeChangePasswordHasher struct {
	verifyCalled bool
	verifyError  error

	verifyPlainPassword string
	verifyPasswordHash  string

	hashCalled bool
	hashError  error

	hashPlainPassword string
	hashResult        string
}

func (f *fakeChangePasswordHasher) Verify(
	ctx context.Context,
	plainPassword string,
	passwordHash string,
) error {
	f.verifyCalled = true
	f.verifyPlainPassword = plainPassword
	f.verifyPasswordHash = passwordHash

	return f.verifyError
}

func (f *fakeChangePasswordHasher) Hash(
	ctx context.Context,
	plainPassword string,
) (string, error) {
	f.hashCalled = true
	f.hashPlainPassword = plainPassword

	if f.hashError != nil {
		return "", f.hashError
	}

	if f.hashResult != "" {
		return f.hashResult, nil
	}

	return "hashed:" + plainPassword, nil
}

// newChangePasswordUser creates a valid active user for the tests.
//
// The stored password represents the hash of:
//
//	OldPassword@123
func newChangePasswordUser(t *testing.T) *user.User {
	t.Helper()

	id := user.NewUserID()

	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("john@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed:OldPassword@123")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	now := time.Now()

	return user.ReconstituteUser(
		id,
		fullName,
		email,
		passwordHash,
		user.RoleUser,
		user.StatusActive,
		now,
		now,
	)
}

// -----------------------------------------------------------------------------
// Happy path
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ChangesPasswordSuccessfully(t *testing.T) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.findByIDCalled {
		t.Fatal("expected FindByID to be called")
	}

	if !hasher.verifyCalled {
		t.Fatal("expected password verification to be called")
	}

	if hasher.verifyPlainPassword != "OldPassword@123" {
		t.Fatalf(
			"expected current plaintext password %q, got %q",
			"OldPassword@123",
			hasher.verifyPlainPassword,
		)
	}

	if hasher.verifyPasswordHash != "hashed:OldPassword@123" {
		t.Fatalf(
			"expected stored password hash %q, got %q",
			"hashed:OldPassword@123",
			hasher.verifyPasswordHash,
		)
	}

	if !hasher.hashCalled {
		t.Fatal("expected new password to be hashed")
	}

	if hasher.hashPlainPassword != "NewPassword@456" {
		t.Fatalf(
			"expected new plaintext password %q, got %q",
			"NewPassword@456",
			hasher.hashPlainPassword,
		)
	}

	if !repository.updatePasswordHashCalled {
		t.Fatal("expected UpdatePasswordHash to be called")
	}

	if repository.updatedUserID != testUser.ID() {
		t.Fatalf(
			"expected updated user ID %s, got %s",
			testUser.ID(),
			repository.updatedUserID,
		)
	}

	if repository.updatedPasswordHash.String() != "hashed:NewPassword@456" {
		t.Fatalf(
			"expected new password hash %q, got %q",
			"hashed:NewPassword@456",
			repository.updatedPasswordHash.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// User identity / lookup failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorForInvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange.
	repository := &fakeChangePasswordRepository{}
	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		"not-a-valid-user-id",
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if err == nil {
		t.Fatal("expected an error for an invalid user ID")
	}

	if repository.findByIDCalled {
		t.Fatal("expected FindByID not to be called")
	}

	if hasher.verifyCalled {
		t.Fatal("expected password verification not to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected password hashing not to be called")
	}
}

func TestChangePasswordUseCase_ReturnsUserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange.
	repository := &fakeChangePasswordRepository{
		user: nil,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		user.NewUserID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if !errors.Is(err, use_cases.ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}

	if hasher.verifyCalled {
		t.Fatal("expected password verification not to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}
}

func TestChangePasswordUseCase_ReturnsRepositoryLookupError(t *testing.T) {
	t.Parallel()

	// Arrange.
	expectedError := errors.New("database lookup failed")

	repository := &fakeChangePasswordRepository{
		user:          newChangePasswordUser(t),
		findByIDError: expectedError,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		repository.user.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected repository error to be propagated, got %v",
			err,
		)
	}

	if hasher.verifyCalled {
		t.Fatal("expected password verification not to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}
}

// -----------------------------------------------------------------------------
// Account-status rules
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_RejectsSuspendedAccount(t *testing.T) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)
	testUser.Suspend()

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if err == nil {
		t.Fatal("expected suspended account to be rejected")
	}

	if hasher.verifyCalled {
		t.Fatal("expected password verification not to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}
}

func TestChangePasswordUseCase_RejectsInactiveAccount(t *testing.T) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)
	testUser.Deactivate()

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if err == nil {
		t.Fatal("expected inactive account to be rejected")
	}

	if hasher.verifyCalled {
		t.Fatal("expected password verification not to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}
}

// -----------------------------------------------------------------------------
// Current-password verification failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenCurrentPasswordIsIncorrect(t *testing.T) {
	t.Parallel()

	// Arrange.
	expectedError := errors.New("invalid current password")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		verifyError: expectedError,
	}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"WrongPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected verification error to be returned, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected password verification to be called")
	}

	// Security invariant:
	// Never hash a new password when the current password has not
	// been successfully verified.
	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to occur")
	}

	// Security invariant:
	// Never persist a password change when verification fails.
	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to occur")
	}
}

func TestChangePasswordUseCase_DoesNotHashOrPersistWhenVerificationFails(
	t *testing.T,
) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		verifyError: errors.New("password mismatch"),
	}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	_ = useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"WrongPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if hasher.hashCalled {
		t.Fatal("expected Hash not to be called after Verify failure")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected UpdatePasswordHash not to be called after Verify failure")
	}
}

// -----------------------------------------------------------------------------
// New-password hashing failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenNewPasswordHashingFails(t *testing.T) {
	t.Parallel()

	// Arrange.
	expectedError := errors.New("password hashing failed")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		hashError: expectedError,
	}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected hashing error to be returned, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected password verification to be called")
	}

	if !hasher.hashCalled {
		t.Fatal("expected password hashing to be called")
	}

	// Security invariant:
	// A password must never be persisted if hashing failed.
	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to occur")
	}
}

func TestChangePasswordUseCase_DoesNotPersistWhenHashingFails(t *testing.T) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		hashError: errors.New("hash generation failed"),
	}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	_ = useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if repository.updatePasswordHashCalled {
		t.Fatal("expected UpdatePasswordHash not to be called when hashing fails")
	}
}

// -----------------------------------------------------------------------------
// Persistence failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenPasswordPersistenceFails(t *testing.T) {
	t.Parallel()

	// Arrange.
	expectedError := errors.New("password persistence failed")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user:                    testUser,
		updatePasswordHashError: expectedError,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected persistence error to be returned, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected password verification to be called")
	}

	if !hasher.hashCalled {
		t.Fatal("expected password hashing to be called")
	}

	if !repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence to be attempted")
	}
}

// -----------------------------------------------------------------------------
// Security invariants
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_NeverSendsPlaintextPasswordToRepository(
	t *testing.T,
) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	currentPassword := "OldPassword@123"
	newPassword := "NewPassword@456"

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		currentPassword,
		newPassword,
	)

	// Assert.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	persistedHash := repository.updatedPasswordHash.String()

	if persistedHash == currentPassword {
		t.Fatal("repository received the current plaintext password")
	}

	if persistedHash == newPassword {
		t.Fatal("repository received the new plaintext password")
	}

	if persistedHash != "hashed:"+newPassword {
		t.Fatalf(
			"expected repository to receive only the hashed password, got %q",
			persistedHash,
		)
	}
}

func TestChangePasswordUseCase_VerifiesAgainstStoredPasswordHash(
	t *testing.T,
) {
	t.Parallel()

	// Arrange.
	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
	)

	// Act.
	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	// Assert.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedStoredHash := "hashed:OldPassword@123"

	if hasher.verifyPasswordHash != expectedStoredHash {
		t.Fatalf(
			"expected Verify to receive stored hash %q, got %q",
			expectedStoredHash,
			hasher.verifyPasswordHash,
		)
	}
}

// -----------------------------------------------------------------------------
// Compile-time interface assertions
// -----------------------------------------------------------------------------

var _ ports.UserRepository = (*fakeChangePasswordRepository)(nil)
var _ ports.PasswordHasher = (*fakeChangePasswordHasher)(nil)
