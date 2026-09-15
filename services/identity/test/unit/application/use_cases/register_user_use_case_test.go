package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// fakeLogger is a test implementation of the Logger port.
//
// Registration observability tests use this fake to capture structured
// application events without depending on the concrete infrastructure
// logging implementation.
type fakeLogger struct {
	events []ports.LogEvent
	err    error
}

func (f *fakeLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)

	return f.err
}

// fakeMetrics is a test implementation of the Metrics port.
//
// Registration observability tests use this fake to capture application
// metrics without depending on the concrete infrastructure metrics
// implementation.
type fakeMetrics struct {
	increments   []ports.Metric
	observations []ports.Metric
	incrementErr error
	observeErr   error
}

func (f *fakeMetrics) Increment(
	_ context.Context,
	metric ports.Metric,
) error {
	f.increments = append(f.increments, metric)

	return f.incrementErr
}

func (f *fakeMetrics) Observe(
	_ context.Context,
	metric ports.Metric,
) error {
	f.observations = append(f.observations, metric)

	return f.observeErr
}

var _ ports.Metrics = (*fakeMetrics)(nil)

// lastEvent returns the most recently recorded log event.
//
// Registration tests use this helper after asserting that an event was
// emitted, keeping the individual tests focused on the event contract.
func (f *fakeLogger) lastEvent(t *testing.T) ports.LogEvent {
	t.Helper()

	if len(f.events) == 0 {
		t.Fatal("expected at least one log event")
	}

	return f.events[len(f.events)-1]
}

// assertRegistrationEvent verifies the common fields that identify a
// registration event.
//
// Registration does not authenticate an existing user, so UserID and Role
// are intentionally only expected on the successful event, where the newly
// created aggregate is available.
func assertRegistrationEvent(
	t *testing.T,
	event ports.LogEvent,
	expectedEvent string,
	expectedOperation string,
	expectedFailureCategory string,
) {
	t.Helper()

	if event.Event != expectedEvent {
		t.Fatalf(
			"expected event %q, got %q",
			expectedEvent,
			event.Event,
		)
	}

	if event.Operation != expectedOperation {
		t.Fatalf(
			"expected operation %q, got %q",
			expectedOperation,
			event.Operation,
		)
	}

	if event.FailureCategory != expectedFailureCategory {
		t.Fatalf(
			"expected failure category %q, got %q",
			expectedFailureCategory,
			event.FailureCategory,
		)
	}
}

// assertEventDoesNotContainSecret verifies that registration observability
// never exposes authentication secrets.
//
// The plaintext password and generated password hash must not be written to
// any structured logging field.
func assertEventDoesNotContainSecret(
	t *testing.T,
	event ports.LogEvent,
	secret string,
) {
	t.Helper()

	if event.Event == secret ||
		event.Operation == secret ||
		event.UserID == secret ||
		event.Role == secret ||
		event.HTTPMethod == secret ||
		event.Route == secret ||
		event.FailureCategory == secret ||
		event.RequiredRole == secret {
		t.Fatalf(
			"log event unexpectedly contained secret %q: %+v",
			secret,
			event,
		)
	}
}

// fakePasswordHasher is a test implementation of the PasswordHasher port.
//
// It allows the registration use case to be tested without depending on a
// real password hashing algorithm.
type fakePasswordHasher struct {
	hashCalled bool
	hashError  error
}

func (f *fakePasswordHasher) Hash(
	_ context.Context,
	plainPassword string,
) (string, error) {
	f.hashCalled = true

	if f.hashError != nil {
		return "", f.hashError
	}

	return "hashed:" + plainPassword, nil
}

// Verify satisfies the PasswordHasher interface.
//
// Registration does not perform password verification, so this method is
// intentionally unused by the registration tests.
func (f *fakePasswordHasher) Verify(
	ctx context.Context,
	plainPassword string,
	passwordHash string,
) error {
	return nil
}

// fakeUserRepository is a test implementation of the UserRepository port.
//
// The fake records the User aggregate passed to Create so tests can inspect
// exactly what the use case attempted to persist.
type fakeUserRepository struct {
	existingEmail bool
	existsError   error
	createError   error
	createdUser   *user.User
}

func (f *fakeUserRepository) ExistsByEmail(
	_ context.Context,
	_ user.Email,
) (bool, error) {
	if f.existsError != nil {
		return false, f.existsError
	}

	return f.existingEmail, nil
}

// FindByID is implemented to satisfy the UserRepository contract.
//
// Registration does not use this operation, so the fake returns nil.
func (f *fakeUserRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeUserRepository) Create(
	_ context.Context,
	newUser *user.User,
) error {
	if f.createError != nil {
		return f.createError
	}

	f.createdUser = newUser

	return nil
}

// UpdateFullName satisfies the UserRepository interface.
func (f *fakeUserRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// UpdateStatus satisfies the UserRepository interface.
//
// These profile-update tests do not exercise account-status persistence.
func (f *fakeUserRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUserRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUserRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
}

// FindByEmail is implemented to satisfy the UserRepository contract.
//
// Registration does not use this operation, so the fake returns nil.
// Authentication tests will provide their own behavior for this method.
func (f *fakeUserRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

func TestRegisterUserUseCase_RegistersUserSuccessfully(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("expected result to contain user ID")
	}

	if result.FullName != "John Doe" {
		t.Fatalf(
			"expected full name %q, got %q",
			"John Doe",
			result.FullName,
		)
	}

	if result.Email != "john@example.com" {
		t.Fatalf(
			"expected email %q, got %q",
			"john@example.com",
			result.Email,
		)
	}

	if result.Role != user.RoleUser.String() {
		t.Fatalf(
			"expected role %q, got %q",
			user.RoleUser.String(),
			result.Role,
		)
	}

	if result.Status != user.StatusPendingVerification.String() {
		t.Fatalf(
			"expected status %q, got %q",
			user.StatusPendingVerification.String(),
			result.Status,
		)
	}

	if !hasher.hashCalled {
		t.Fatal("expected password hasher to be called")
	}

	if repository.createdUser == nil {
		t.Fatal("expected user to be created")
	}

	if repository.createdUser.PasswordHash().String() != "hashed:StrongPass@123" {
		t.Fatal("expected persisted user to contain the hashed password")
	}
}

func TestRegisterUserUseCase_DoesNotRegisterDuplicateEmail(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{
		existingEmail: true,
	}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, application_errors.ErrEmailAlreadyExists) {
		t.Fatalf(
			"expected ErrEmailAlreadyExists, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for a duplicate email",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected duplicate user not to be created")
	}
}

func TestRegisterUserUseCase_ReturnsEmailValidationError(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "invalid-email",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err == nil {
		t.Fatal("expected invalid email to return an error")
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur after email validation fails",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_ReturnsRepositoryExistsError(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("failed to check existing email")

	repository := &fakeUserRepository{
		existsError: expectedErr,
	}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error %v, got %v",
			expectedErr,
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur when email lookup fails",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected user not to be created")
	}
}

func TestRegisterUserUseCase_ReturnsPasswordHasherError(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("password hashing failed")

	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{
		hashError: expectedErr,
	}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected password hasher error %v, got %v",
			expectedErr,
			err,
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected user not to be created")
	}
}

func TestRegisterUserUseCase_ReturnsFullNameValidationError(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err == nil {
		t.Fatal("expected invalid full name to return an error")
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_ReturnsRepositoryCreateError(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("failed to create user")

	repository := &fakeUserRepository{
		createError: expectedErr,
	}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository create error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestRegisterUserUseCase_DoesNotExposePasswordHash(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	if repository.createdUser == nil {
		t.Fatal("expected user to be created")
	}

	// The domain aggregate contains the password hash because it is needed
	// later for authentication. RegisterUserResult deliberately exposes no
	// password or password hash.
	if repository.createdUser.PasswordHash().String() == "" {
		t.Fatal("expected persisted user to contain a password hash")
	}

	if result.Email != command.Email {
		t.Fatalf(
			"expected result email %q, got %q",
			command.Email,
			result.Email,
		)
	}
}

func TestRegisterUserUseCase_RejectsPasswordShorterThanEightCharacters(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "Abc@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordTooShort) {
		t.Fatalf(
			"expected ErrPasswordTooShort, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_RejectsPasswordLongerThanSixtyFourCharacters(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	password := "A" +
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz1234567890@x"

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: password,
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordTooLong) {
		t.Fatalf(
			"expected ErrPasswordTooLong, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_RejectsPasswordMissingUppercase(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "strong@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordMissingUppercase) {
		t.Fatalf(
			"expected ErrPasswordMissingUppercase, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_RejectsPasswordMissingLowercase(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "STRONG@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordMissingLowercase) {
		t.Fatalf(
			"expected ErrPasswordMissingLowercase, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_RejectsPasswordMissingNumber(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPassword@",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordMissingNumber) {
		t.Fatalf(
			"expected ErrPasswordMissingNumber, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

func TestRegisterUserUseCase_RejectsPasswordMissingSpecialCharacter(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPassword123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordMissingSpecialCharacter) {
		t.Fatalf(
			"expected ErrPasswordMissingSpecialCharacter, got %v",
			err,
		)
	}

	if hasher.hashCalled {
		t.Fatal(
			"expected password hashing not to occur for an invalid password",
		)
	}

	if repository.createdUser != nil {
		t.Fatal("expected invalid user not to be created")
	}
}

// -----------------------------------------------------------------------------
// Registration observability tests
// -----------------------------------------------------------------------------

func TestRegisterUserUseCase_LogsSuccessfulRegistration(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("expected successful registration to return a user ID")
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.succeeded",
		"registration",
		"",
	)

	if event.UserID != result.ID {
		t.Fatalf(
			"expected event user ID %q, got %q",
			result.ID,
			event.UserID,
		)
	}

	if event.Role != user.RoleUser.String() {
		t.Fatalf(
			"expected event role %q, got %q",
			user.RoleUser.String(),
			event.Role,
		)
	}
}

func TestRegisterUserUseCase_LogsDuplicateEmail(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{
		existingEmail: true,
	}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, application_errors.ErrEmailAlreadyExists) {
		t.Fatalf(
			"expected ErrEmailAlreadyExists, got %v",
			err,
		)
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.duplicate_email",
		"registration",
		"duplicate_email",
	)

	if event.UserID != "" {
		t.Fatalf(
			"expected duplicate email event to have no user ID, got %q",
			event.UserID,
		)
	}
}

func TestRegisterUserUseCase_LogsEmailValidationFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "invalid-email",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err == nil {
		t.Fatal("expected invalid email to return an error")
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.validation_failed",
		"registration",
		"validation_failed",
	)
}

func TestRegisterUserUseCase_LogsPasswordValidationFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "weak",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, policies.ErrPasswordTooShort) {
		t.Fatalf(
			"expected ErrPasswordTooShort, got %v",
			err,
		)
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.validation_failed",
		"registration",
		"validation_failed",
	)
}

func TestRegisterUserUseCase_LogsRepositoryExistsFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("failed to check existing email")

	repository := &fakeUserRepository{
		existsError: expectedErr,
	}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error %v, got %v",
			expectedErr,
			err,
		)
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.failed",
		"registration",
		"email_existence_check_failed",
	)
}

func TestRegisterUserUseCase_LogsPasswordHashingFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("password hashing failed")

	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{
		hashError: expectedErr,
	}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected password hasher error %v, got %v",
			expectedErr,
			err,
		)
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.failed",
		"registration",
		"password_hashing_failed",
	)
}

func TestRegisterUserUseCase_LogsFullNameValidationFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err == nil {
		t.Fatal("expected invalid full name to return an error")
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.validation_failed",
		"registration",
		"validation_failed",
	)
}

func TestRegisterUserUseCase_LogsRepositoryCreateFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("failed to create user")

	repository := &fakeUserRepository{
		createError: expectedErr,
	}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository create error %v, got %v",
			expectedErr,
			err,
		)
	}

	event := logger.lastEvent(t)

	assertRegistrationEvent(
		t,
		event,
		"auth.registration.failed",
		"registration",
		"user_creation_failed",
	)
}

func TestRegisterUserUseCase_LoggerFailureDoesNotAffectRegistration(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{
		err: errors.New("logger unavailable"),
	}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed even when logging fails, got: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("expected successful registration to return a user ID")
	}

	if repository.createdUser == nil {
		t.Fatal("expected user to be created")
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected one attempted log event, got %d",
			len(logger.events),
		)
	}
}

func TestRegisterUserUseCase_DoesNotLogRegistrationSecrets(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	logger := &fakeLogger{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		logger,
		nil,
	)

	const plaintextPassword = "StrongPass@123"
	const passwordHash = "hashed:" + plaintextPassword

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: plaintextPassword,
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	event := logger.lastEvent(t)

	assertEventDoesNotContainSecret(
		t,
		event,
		plaintextPassword,
	)

	assertEventDoesNotContainSecret(
		t,
		event,
		passwordHash,
	)
}
func TestRegisterUserUseCase_RecordsSuccessfulRegistrationMetrics(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	metrics := &fakeMetrics{}

	// Metrics are not wired into the current implementation yet.
	// This test intentionally drives the TDD change.
	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		metrics,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("expected successful registration to return a user ID")
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected one registration counter metric, got %d",
			len(metrics.increments),
		)
	}

	counter := metrics.increments[0]

	if counter.Name != "auth.registration" {
		t.Fatalf(
			"expected metric %q, got %q",
			"auth.registration",
			counter.Name,
		)
	}

	if counter.Value != 1 {
		t.Fatalf(
			"expected metric value 1, got %v",
			counter.Value,
		)
	}

	if counter.Labels["result"] != "success" {
		t.Fatalf(
			"expected result label %q, got %q",
			"success",
			counter.Labels["result"],
		)
	}

	if len(counter.Labels) != 1 {
		t.Fatalf(
			"expected only the low-cardinality result label, got %+v",
			counter.Labels,
		)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf(
			"expected one registration duration metric, got %d",
			len(metrics.observations),
		)
	}

	duration := metrics.observations[0]

	if duration.Name != "auth.registration.duration" {
		t.Fatalf(
			"expected metric %q, got %q",
			"auth.registration.duration",
			duration.Name,
		)
	}

	if duration.Value < 0 {
		t.Fatalf(
			"expected non-negative registration duration, got %v",
			duration.Value,
		)
	}
}

func TestRegisterUserUseCase_RecordsFailedRegistrationMetrics(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{
		existingEmail: true,
	}
	hasher := &fakePasswordHasher{}
	metrics := &fakeMetrics{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		metrics,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if !errors.Is(err, application_errors.ErrEmailAlreadyExists) {
		t.Fatalf(
			"expected ErrEmailAlreadyExists, got %v",
			err,
		)
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected one registration counter metric, got %d",
			len(metrics.increments),
		)
	}

	counter := metrics.increments[0]

	if counter.Name != "auth.registration" {
		t.Fatalf(
			"expected metric %q, got %q",
			"auth.registration",
			counter.Name,
		)
	}

	if counter.Value != 1 {
		t.Fatalf(
			"expected metric value 1, got %v",
			counter.Value,
		)
	}

	if counter.Labels["result"] != "failure" {
		t.Fatalf(
			"expected result label %q, got %q",
			"failure",
			counter.Labels["result"],
		)
	}

	if len(counter.Labels) != 1 {
		t.Fatalf(
			"expected only the low-cardinality result label, got %+v",
			counter.Labels,
		)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf(
			"expected one registration duration metric, got %d",
			len(metrics.observations),
		)
	}

	if metrics.observations[0].Value < 0 {
		t.Fatalf(
			"expected non-negative registration duration, got %v",
			metrics.observations[0].Value,
		)
	}
}

func TestRegisterUserUseCase_MetricsFailureDoesNotAffectRegistration(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	metrics := &fakeMetrics{
		incrementErr: errors.New("metrics increment unavailable"),
		observeErr:   errors.New("metrics observation unavailable"),
	}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		metrics,
	)

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "StrongPass@123",
	}

	// Act
	result, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed when metrics fail, got: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("expected successful registration to return a user ID")
	}

	if repository.createdUser == nil {
		t.Fatal("expected user to be created")
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected one attempted counter metric, got %d",
			len(metrics.increments),
		)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf(
			"expected one attempted duration metric, got %d",
			len(metrics.observations),
		)
	}
}

func TestRegisterUserUseCase_DoesNotUseHighCardinalityRegistrationMetricLabels(
	t *testing.T,
) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}
	metrics := &fakeMetrics{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
		nil,
		metrics,
	)

	const email = "john@example.com"
	const password = "StrongPass@123"

	command := dto.RegisterUserCommand{
		FullName: "John Doe",
		Email:    email,
		Password: password,
	}

	// Act
	_, err := useCase.Execute(
		context.Background(),
		command,
	)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected registration to succeed, got: %v",
			err,
		)
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected one registration counter metric, got %d",
			len(metrics.increments),
		)
	}

	metric := metrics.increments[0]

	if len(metric.Labels) != 1 {
		t.Fatalf(
			"expected exactly one low-cardinality label, got %+v",
			metric.Labels,
		)
	}

	if metric.Labels["result"] != "success" {
		t.Fatalf(
			"expected result label %q, got %q",
			"success",
			metric.Labels["result"],
		)
	}

	for key, value := range metric.Labels {
		if value == email ||
			value == password ||
			value == "John Doe" ||
			value == resultUserID(repository) {
			t.Fatalf(
				"metric label %q=%q unexpectedly contains high-cardinality or sensitive data",
				key,
				value,
			)
		}
	}
}

func resultUserID(repository *fakeUserRepository) string {
	if repository.createdUser == nil {
		return ""
	}

	return repository.createdUser.ID().String()
}
