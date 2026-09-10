package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// -----------------------------------------------------------------------------
// Test doubles
// -----------------------------------------------------------------------------

type fakeChangePasswordLogger struct {
	events []ports.LogEvent
	err    error
}

func (f *fakeChangePasswordLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)
	return f.err
}

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

// UpdateStatus satisfies the UserRepository interface.
//
// These tests do not exercise account-status persistence.
func (f *fakeChangePasswordRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeChangePasswordRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
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

func assertSinglePasswordChangeEvent(
	t *testing.T,
	logger *fakeChangePasswordLogger,
	expectedEvent string,
	expectedFailure string,
	expectedUserID string,
	expectedRole string,
) {
	t.Helper()

	if len(logger.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(logger.events))
	}

	event := logger.events[0]

	if event.Event != expectedEvent {
		t.Fatalf(
			"expected event %q, got %q",
			expectedEvent,
			event.Event,
		)
	}

	if event.Operation != "change_password" {
		t.Fatalf(
			"expected operation %q, got %q",
			"change_password",
			event.Operation,
		)
	}

	if event.UserID != expectedUserID {
		t.Fatalf(
			"expected user ID %q, got %q",
			expectedUserID,
			event.UserID,
		)
	}

	if event.Role != expectedRole {
		t.Fatalf(
			"expected role %q, got %q",
			expectedRole,
			event.Role,
		)
	}

	if event.FailureCategory != expectedFailure {
		t.Fatalf(
			"expected failure category %q, got %q",
			expectedFailure,
			event.FailureCategory,
		)
	}
}

// -----------------------------------------------------------------------------
// Happy path
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ChangesPasswordSuccessfully(t *testing.T) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	before := testUser.UpdatedAt()

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

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

	if testUser.UpdatedAt().Before(before) {
		t.Fatal("expected domain to update UpdatedAt")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.succeeded",
		"",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

// -----------------------------------------------------------------------------
// User identity / lookup failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorForInvalidUserID(t *testing.T) {
	t.Parallel()

	repository := &fakeChangePasswordRepository{}
	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		"not-a-valid-user-id",
		"OldPassword@123",
		"NewPassword@456",
	)

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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		"",
		"",
	)
}

func TestChangePasswordUseCase_ReturnsUserNotFound(t *testing.T) {
	t.Parallel()

	repository := &fakeChangePasswordRepository{
		user: nil,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	userID := user.NewUserID().String()

	err := useCase.Execute(
		context.Background(),
		userID,
		"OldPassword@123",
		"NewPassword@456",
	)

	if !errors.Is(err, application_errors.ErrUserNotFound) {
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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.not_found",
		"user_not_found",
		userID,
		"",
	)
}

func TestChangePasswordUseCase_ReturnsRepositoryLookupError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("database lookup failed")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user:          testUser,
		findByIDError: expectedError,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.failed",
		"user_lookup_failed",
		testUser.ID().String(),
		"",
	)
}

// -----------------------------------------------------------------------------
// Account-status rules
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_RejectsSuspendedAccount(t *testing.T) {
	t.Parallel()

	testUser := newChangePasswordUser(t)
	if err := testUser.Suspend(); err != nil {
		t.Fatalf("failed to suspend test user: %v", err)
	}

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	if !errors.Is(err, application_errors.ErrAccountNotActive) {
		t.Fatalf(
			"expected ErrAccountNotActive, got %v",
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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.user_inactive",
		"user_inactive",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsInactiveAccount(t *testing.T) {
	t.Parallel()

	testUser := newChangePasswordUser(t)
	if err := testUser.Deactivate(); err != nil {
		t.Fatalf("failed to deactivate test user: %v", err)
	}

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	if !errors.Is(err, application_errors.ErrAccountNotActive) {
		t.Fatalf(
			"expected ErrAccountNotActive, got %v",
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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.user_inactive",
		"user_inactive",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

// -----------------------------------------------------------------------------
// Current-password verification failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenCurrentPasswordIsIncorrect(
	t *testing.T,
) {
	t.Parallel()

	expectedError := errors.New("invalid current password")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		verifyError: expectedError,
	}

	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"WrongPassword@123",
		"NewPassword@456",
	)

	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected verification error to be returned, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to occur")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to occur")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.invalid_credentials",
		"invalid_credentials",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_DoesNotHashOrPersistWhenVerificationFails(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		verifyError: errors.New("password mismatch"),
	}

	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	_ = useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"WrongPassword@123",
		"NewPassword@456",
	)

	if hasher.hashCalled {
		t.Fatal("expected Hash not to be called after Verify failure")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected UpdatePasswordHash not to be called after Verify failure")
	}
}

// -----------------------------------------------------------------------------
// New-password policy rules
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_RejectsNewPasswordShorterThanEightCharacters(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"Abc@123",
	)

	if !errors.Is(err, policies.ErrPasswordTooShort) {
		t.Fatalf(
			"expected ErrPasswordTooShort, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsNewPasswordLongerThanSixtyFourCharacters(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	password := "A" + "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz1234567890@x"

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		password,
	)

	if !errors.Is(err, policies.ErrPasswordTooLong) {
		t.Fatalf(
			"expected ErrPasswordTooLong, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsNewPasswordMissingUppercaseLetter(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"strong@123",
	)

	if !errors.Is(err, policies.ErrPasswordMissingUppercase) {
		t.Fatalf(
			"expected ErrPasswordMissingUppercase, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsNewPasswordMissingLowercaseLetter(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"STRONG@123",
	)

	if !errors.Is(err, policies.ErrPasswordMissingLowercase) {
		t.Fatalf(
			"expected ErrPasswordMissingLowercase, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsNewPasswordMissingNumber(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"StrongPassword@",
	)

	if !errors.Is(err, policies.ErrPasswordMissingNumber) {
		t.Fatalf(
			"expected ErrPasswordMissingNumber, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_RejectsNewPasswordMissingSpecialCharacter(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"StrongPassword123",
	)

	if !errors.Is(err, policies.ErrPasswordMissingSpecialCharacter) {
		t.Fatalf(
			"expected ErrPasswordMissingSpecialCharacter, got %v",
			err,
		)
	}

	if !hasher.verifyCalled {
		t.Fatal("expected current password verification to be called")
	}

	if hasher.hashCalled {
		t.Fatal("expected new password hashing not to be called")
	}

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to be called")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.validation_failed",
		"validation_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

// -----------------------------------------------------------------------------
// New-password hashing failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenNewPasswordHashingFails(
	t *testing.T,
) {
	t.Parallel()

	expectedError := errors.New("password hashing failed")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		hashError: expectedError,
	}

	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

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

	if repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence not to occur")
	}

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.failed",
		"password_hashing_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

func TestChangePasswordUseCase_DoesNotPersistWhenHashingFails(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{
		hashError: errors.New("hash generation failed"),
	}

	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	_ = useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	if repository.updatePasswordHashCalled {
		t.Fatal("expected UpdatePasswordHash not to be called when hashing fails")
	}
}

// -----------------------------------------------------------------------------
// Persistence failures
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_ReturnsErrorWhenPasswordPersistenceFails(
	t *testing.T,
) {
	t.Parallel()

	expectedError := errors.New("password persistence failed")

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user:                    testUser,
		updatePasswordHashError: expectedError,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

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

	assertSinglePasswordChangeEvent(
		t,
		logger,
		"auth.password_change.failed",
		"password_update_failed",
		testUser.ID().String(),
		string(testUser.Role()),
	)
}

// -----------------------------------------------------------------------------
// Observability reliability
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_LoggerFailureDoesNotAffectSuccess(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}

	logger := &fakeChangePasswordLogger{
		err: errors.New("logger unavailable"),
	}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

	if err != nil {
		t.Fatalf(
			"expected logger failure to be ignored, got %v",
			err,
		)
	}

	if !repository.updatePasswordHashCalled {
		t.Fatal("expected password persistence to be called")
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 attempted log event, got %d",
			len(logger.events),
		)
	}
}

// -----------------------------------------------------------------------------
// Security invariants
// -----------------------------------------------------------------------------

func TestChangePasswordUseCase_NeverSendsPlaintextPasswordToRepository(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	currentPassword := "OldPassword@123"
	newPassword := "NewPassword@456"

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		currentPassword,
		newPassword,
	)

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

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		"OldPassword@123",
		"NewPassword@456",
	)

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

func TestChangePasswordUseCase_DoesNotLogPasswordsOrPasswordHashes(
	t *testing.T,
) {
	t.Parallel()

	testUser := newChangePasswordUser(t)

	repository := &fakeChangePasswordRepository{
		user: testUser,
	}

	hasher := &fakeChangePasswordHasher{}
	logger := &fakeChangePasswordLogger{}

	useCase := use_cases.NewChangePasswordUseCase(
		repository,
		hasher,
		logger,
	)

	currentPassword := "OldPassword@123"
	newPassword := "NewPassword@456"

	err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		currentPassword,
		newPassword,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(logger.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(logger.events))
	}

	event := logger.events[0]

	if event.UserID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			event.UserID,
		)
	}

	if event.Role != string(testUser.Role()) {
		t.Fatalf(
			"expected role %q, got %q",
			testUser.Role(),
			event.Role,
		)
	}

	if event.Route != "" {
		t.Fatalf(
			"expected route to be empty at application layer, got %q",
			event.Route,
		)
	}

	if event.HTTPMethod != "" {
		t.Fatalf(
			"expected HTTP method to be empty at application layer, got %q",
			event.HTTPMethod,
		)
	}

	if event.FailureCategory != "" {
		t.Fatalf(
			"expected no failure category on success, got %q",
			event.FailureCategory,
		)
	}
}

// -----------------------------------------------------------------------------
// Compile-time interface assertions
// -----------------------------------------------------------------------------

var _ ports.UserRepository = (*fakeChangePasswordRepository)(nil)
var _ ports.PasswordHasher = (*fakeChangePasswordHasher)(nil)
var _ ports.Logger = (*fakeChangePasswordLogger)(nil)
