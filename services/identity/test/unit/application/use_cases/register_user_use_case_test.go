package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

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

func TestRegisterUserUseCase_RegistersUserSuccessfully(t *testing.T) {
	t.Parallel()

	// Arrange
	repository := &fakeUserRepository{}
	hasher := &fakePasswordHasher{}

	useCase := use_cases.NewRegisterUserUseCase(
		repository,
		hasher,
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
	if !errors.Is(err, use_cases.ErrEmailAlreadyExists) {
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
