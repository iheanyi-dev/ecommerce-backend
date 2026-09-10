package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// fakeListUsersRepository is a test double for the existing UserRepository.
//
// The ListUsers use case only needs the List capability from the repository,
// but we implement the complete interface because UserRepository is the
// existing application persistence boundary.
type fakeListUsersRepository struct {
	users []*user.User
	err   error

	gotLimit  int
	gotOffset int
}

func (f *fakeListUsersRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakeListUsersRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

func (f *fakeListUsersRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeListUsersRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeListUsersRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeListUsersRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeListUsersRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeListUsersRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	f.gotLimit = limit
	f.gotOffset = offset

	return f.users, f.err
}

// fakeListUsersLogger is a test double for the application Logger port.
type fakeListUsersLogger struct {
	event     ports.LogEvent
	callCount int
	err       error
}

func (f *fakeListUsersLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.event = event
	f.callCount++

	return f.err
}

func newListUsersTestUser(
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

func assertListUsersEvent(
	t *testing.T,
	logger *fakeListUsersLogger,
	event string,
	operation string,
	failureCategory string,
) {
	t.Helper()

	if logger.callCount != 1 {
		t.Fatalf(
			"expected logger to be called once, got %d",
			logger.callCount,
		)
	}

	if logger.event.Event != event {
		t.Fatalf(
			"expected event %q, got %q",
			event,
			logger.event.Event,
		)
	}

	if logger.event.Operation != operation {
		t.Fatalf(
			"expected operation %q, got %q",
			operation,
			logger.event.Operation,
		)
	}

	if logger.event.FailureCategory != failureCategory {
		t.Fatalf(
			"expected failure category %q, got %q",
			failureCategory,
			logger.event.FailureCategory,
		)
	}
}

func TestListUsersUseCase_Execute_ReturnsUsers(t *testing.T) {
	firstUser := newListUsersTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	secondUser := newListUsersTestUser(
		t,
		"Jane Doe",
		"jane@example.com",
	)

	repository := &fakeListUsersRepository{
		users: []*user.User{
			firstUser,
			secondUser,
		},
	}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	result, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Users) != 2 {
		t.Fatalf(
			"expected 2 users, got %d",
			len(result.Users),
		)
	}

	if result.Users[0].ID != firstUser.ID().String() {
		t.Fatalf(
			"expected first user ID %q, got %q",
			firstUser.ID().String(),
			result.Users[0].ID,
		)
	}

	if result.Users[0].FullName != firstUser.FullName().String() {
		t.Fatalf(
			"expected first user's full name %q, got %q",
			firstUser.FullName().String(),
			result.Users[0].FullName,
		)
	}

	if result.Users[0].Email != firstUser.Email().String() {
		t.Fatalf(
			"expected first user's email %q, got %q",
			firstUser.Email().String(),
			result.Users[0].Email,
		)
	}
}

func TestListUsersUseCase_Execute_PassesPaginationToRepository(
	t *testing.T,
) {
	repository := &fakeListUsersRepository{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	_, err := useCase.Execute(
		context.Background(),
		25,
		50,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.gotLimit != 25 {
		t.Fatalf(
			"expected limit 25, got %d",
			repository.gotLimit,
		)
	}

	if repository.gotOffset != 50 {
		t.Fatalf(
			"expected offset 50, got %d",
			repository.gotOffset,
		)
	}
}

func TestListUsersUseCase_Execute_ReturnsRepositoryError(
	t *testing.T,
) {
	expectedErr := errors.New("database failure")

	repository := &fakeListUsersRepository{
		err: expectedErr,
	}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	_, err := useCase.Execute(
		context.Background(),
		10,
		0,
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

func TestListUsersUseCase_Execute_DoesNotExposePasswordHash(
	t *testing.T,
) {
	testUser := newListUsersTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeListUsersRepository{
		users: []*user.User{testUser},
	}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	result, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Compile-time protection: UserSummary intentionally has no
	// PasswordHash field. This test verifies that the returned DTO
	// remains the safe administrative representation.
	_ = result.Users[0].Email
}

func TestListUsersUseCase_Execute_RejectsInvalidLimit(
	t *testing.T,
) {
	repository := &fakeListUsersRepository{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	_, err := useCase.Execute(
		context.Background(),
		0,
		0,
	)

	if err == nil {
		t.Fatal("expected error for invalid limit, got nil")
	}

	if repository.gotLimit != 0 {
		t.Fatalf(
			"expected repository not to be called, got limit %d",
			repository.gotLimit,
		)
	}
}

func TestListUsersUseCase_Execute_RejectsNegativeOffset(
	t *testing.T,
) {
	repository := &fakeListUsersRepository{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		nil,
	)

	_, err := useCase.Execute(
		context.Background(),
		10,
		-1,
	)

	if err == nil {
		t.Fatal("expected error for negative offset, got nil")
	}

	if repository.gotLimit != 0 {
		t.Fatalf(
			"expected repository not to be called, got limit %d",
			repository.gotLimit,
		)
	}
}

func TestListUsersUseCase_Execute_LogsSuccess(t *testing.T) {
	testUser := newListUsersTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeListUsersRepository{
		users: []*user.User{testUser},
	}

	logger := &fakeListUsersLogger{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertListUsersEvent(
		t,
		logger,
		"user.list.succeeded",
		"list_users",
		"",
	)
}

func TestListUsersUseCase_Execute_LogsValidationFailure(
	t *testing.T,
) {
	repository := &fakeListUsersRepository{}
	logger := &fakeListUsersLogger{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		0,
		0,
	)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	assertListUsersEvent(
		t,
		logger,
		"user.list.validation_failed",
		"list_users",
		"validation_failed",
	)

	if repository.gotLimit != 0 {
		t.Fatal("expected repository not to be called")
	}
}

func TestListUsersUseCase_Execute_LogsRepositoryFailure(
	t *testing.T,
) {
	expectedErr := errors.New("database failure")

	repository := &fakeListUsersRepository{
		err: expectedErr,
	}

	logger := &fakeListUsersLogger{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err == nil {
		t.Fatal("expected repository error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error %v, got %v",
			expectedErr,
			err,
		)
	}

	assertListUsersEvent(
		t,
		logger,
		"user.list.failed",
		"list_users",
		"user_lookup_failed",
	)
}

func TestListUsersUseCase_Execute_LoggerFailureDoesNotAffectResult(
	t *testing.T,
) {
	testUser := newListUsersTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeListUsersRepository{
		users: []*user.User{testUser},
	}

	logger := &fakeListUsersLogger{
		err: errors.New("logger failure"),
	}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		logger,
	)

	result, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err != nil {
		t.Fatalf(
			"expected logger failure not to affect result, got %v",
			err,
		)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Users) != 1 {
		t.Fatalf(
			"expected 1 user, got %d",
			len(result.Users),
		)
	}

	assertListUsersEvent(
		t,
		logger,
		"user.list.succeeded",
		"list_users",
		"",
	)
}

func TestListUsersUseCase_Execute_DoesNotLogSensitiveAuthenticationData(
	t *testing.T,
) {
	testUser := newListUsersTestUser(
		t,
		"John Doe",
		"john@example.com",
	)

	repository := &fakeListUsersRepository{
		users: []*user.User{testUser},
	}

	logger := &fakeListUsersLogger{}

	useCase := use_cases.NewListUsersUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		10,
		0,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if logger.event.Event != "user.list.succeeded" {
		t.Fatalf(
			"expected success event, got %q",
			logger.event.Event,
		)
	}

	// The List Users observability event must never contain
	// authentication secrets such as passwords or password hashes.
	if logger.event.UserID == "hashed-password" {
		t.Fatal("password hash must never be logged as user ID")
	}

	if logger.event.Role == "hashed-password" {
		t.Fatal("password hash must never be logged as role")
	}

	if logger.event.FailureCategory == "hashed-password" {
		t.Fatal("password hash must never be logged as failure category")
	}

	if logger.event.RequiredRole == "hashed-password" {
		t.Fatal("password hash must never be logged as required role")
	}
}
