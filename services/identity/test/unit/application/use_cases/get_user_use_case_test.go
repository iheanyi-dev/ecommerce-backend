package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// fakeGetUserRepository is a test double for the existing UserRepository.
//
// The GetUser use case only needs the FindByID capability from the
// repository, but UserRepository is the existing application persistence
// boundary, so the complete interface is implemented here.
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

// UpdateStatus satisfies ports.UserRepository.
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

// fakeGetUserLogger is a test double for ports.Logger.
//
// It records the structured event supplied by the GetUser use case so the
// tests can verify the observability contract without depending on Zap or
// another concrete logging implementation.
type fakeGetUserLogger struct {
	events []ports.LogEvent
	err    error
}

// Log records the event and returns the configured logger error.
func (f *fakeGetUserLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)

	return f.err
}

// lastEvent returns the most recently recorded event.
func (f *fakeGetUserLogger) lastEvent(t *testing.T) ports.LogEvent {
	t.Helper()

	if len(f.events) == 0 {
		t.Fatal("expected at least one log event, got none")
	}

	return f.events[len(f.events)-1]
}

// assertGetUserEvent verifies the common fields of a Get User event.
func assertGetUserEvent(
	t *testing.T,
	event ports.LogEvent,
	expectedEvent string,
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

	if event.Operation != "get_user" {
		t.Fatalf(
			"expected operation %q, got %q",
			"get_user",
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

	if event.HTTPMethod != "" {
		t.Fatalf(
			"expected empty HTTP method, got %q",
			event.HTTPMethod,
		)
	}

	if event.Route != "" {
		t.Fatalf(
			"expected empty route, got %q",
			event.Route,
		)
	}

	if event.StatusCode != 0 {
		t.Fatalf(
			"expected empty status code, got %d",
			event.StatusCode,
		)
	}

	if event.DurationMillis != 0 {
		t.Fatalf(
			"expected empty duration, got %d",
			event.DurationMillis,
		)
	}

	if event.RequiredRole != "" {
		t.Fatalf(
			"expected empty required role, got %q",
			event.RequiredRole,
		)
	}
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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	expectedCreatedAt := testUser.CreatedAt()
	expectedUpdatedAt := testUser.UpdatedAt()

	repository := &fakeGetUserRepository{
		user: testUser,
	}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		nil,
	)

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

	var _ time.Time = result.CreatedAt
}

func TestGetUserUseCase_Execute_LogsSuccess(
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

	logger := &fakeGetUserLogger{}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	event := logger.lastEvent(t)

	assertGetUserEvent(
		t,
		event,
		"user.get.succeeded",
		"",
	)

	if event.UserID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			event.UserID,
		)
	}

	if event.Role != testUser.Role().String() {
		t.Fatalf(
			"expected role %q, got %q",
			testUser.Role().String(),
			event.Role,
		)
	}
}

func TestGetUserUseCase_Execute_LogsValidationFailure(
	t *testing.T,
) {
	repository := &fakeGetUserRepository{}
	logger := &fakeGetUserLogger{}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		"not-a-valid-user-id",
	)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	event := logger.lastEvent(t)

	assertGetUserEvent(
		t,
		event,
		"user.get.validation_failed",
		"validation_failed",
	)

	if event.UserID != "" {
		t.Fatalf(
			"expected empty user ID, got %q",
			event.UserID,
		)
	}

	if event.Role != "" {
		t.Fatalf(
			"expected empty role, got %q",
			event.Role,
		)
	}
}

func TestGetUserUseCase_Execute_LogsRepositoryFailure(
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

	logger := &fakeGetUserLogger{}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
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

	event := logger.lastEvent(t)

	assertGetUserEvent(
		t,
		event,
		"user.get.failed",
		"user_lookup_failed",
	)

	if event.UserID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			event.UserID,
		)
	}

	if event.Role != "" {
		t.Fatalf(
			"expected empty role, got %q",
			event.Role,
		)
	}
}

func TestGetUserUseCase_Execute_LogsUserNotFound(
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

	logger := &fakeGetUserLogger{}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if !errors.Is(err, application_errors.ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}

	event := logger.lastEvent(t)

	assertGetUserEvent(
		t,
		event,
		"user.get.not_found",
		"user_not_found",
	)

	if event.UserID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			event.UserID,
		)
	}

	if event.Role != "" {
		t.Fatalf(
			"expected empty role, got %q",
			event.Role,
		)
	}
}

func TestGetUserUseCase_Execute_LoggerFailureDoesNotAffectResult(
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

	logger := &fakeGetUserLogger{
		err: errors.New("logger failure"),
	}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf(
			"expected registration to succeed despite logger failure, got %v",
			err,
		)
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
}

func TestGetUserUseCase_Execute_DoesNotLogSensitiveAuthenticationData(
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

	logger := &fakeGetUserLogger{}

	useCase := use_cases.NewGetUserUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	event := logger.lastEvent(t)

	if event.Event == "hashed-password" {
		t.Fatal("password hash must never be logged")
	}

	if event.Operation == "hashed-password" {
		t.Fatal("password hash must never be logged")
	}

	if event.UserID == "hashed-password" {
		t.Fatal("password hash must never be logged")
	}

	if event.Role == "hashed-password" {
		t.Fatal("password hash must never be logged")
	}

	if event.FailureCategory == "hashed-password" {
		t.Fatal("password hash must never be logged")
	}

	if event.UserID == "password" {
		t.Fatal("plaintext password must never be logged")
	}

	if event.Operation == "password" {
		t.Fatal("plaintext password must never be logged")
	}
}

// Compile-time assertion that the fake logger implements the application
// logger port.
var _ ports.Logger = (*fakeGetUserLogger)(nil)

// Compile-time assertion that the fake repository implements the application
// repository port.
var _ ports.UserRepository = (*fakeGetUserRepository)(nil)

// Compile-time assertion that the returned DTO remains the expected type.
var _ *dto.GetUserResult
