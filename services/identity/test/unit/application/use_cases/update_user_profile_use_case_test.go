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

// fakeUpdateProfileRepository is an in-memory repository used to test the
// profile update use case without requiring PostgreSQL.
type fakeUpdateProfileRepository struct {
	user *user.User

	findByIDErr       error
	updateFullNameErr error

	updateFullNameCalled bool
	updatedUserID        user.UserID
	updatedFullName      user.FullName
}

func (f *fakeUpdateProfileRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	f.updateFullNameCalled = true
	f.updatedUserID = existingUser.ID()
	f.updatedFullName = existingUser.FullName()

	return f.updateFullNameErr
}

func (f *fakeUpdateProfileRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateProfileRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateProfileRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
}

func (f *fakeUpdateProfileRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakeUpdateProfileRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateProfileRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeUpdateProfileRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	return f.user, nil
}

// fakeUpdateProfileLogger is a test double for the application Logger port.
type fakeUpdateProfileLogger struct {
	event     ports.LogEvent
	callCount int
	err       error
}

func (f *fakeUpdateProfileLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.event = event
	f.callCount++

	return f.err
}

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

func assertUpdateProfileEvent(
	t *testing.T,
	logger *fakeUpdateProfileLogger,
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

// TestUpdateUserProfileUseCase_UpdateFullName verifies that an authenticated
// user can update their full name while immutable account fields remain
// unchanged.
func TestUpdateUserProfileUseCase_UpdateFullName(t *testing.T) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		nil,
	)

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
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			result.UserID,
		)
	}

	if result.FullName != "Updated Name" {
		t.Fatalf(
			"expected full name %q, got %q",
			"Updated Name",
			result.FullName,
		)
	}

	if result.Email != "profile@example.com" {
		t.Fatalf(
			"expected immutable email %q, got %q",
			"profile@example.com",
			result.Email,
		)
	}

	if !repository.updateFullNameCalled {
		t.Fatal("expected UpdateFullName to be called")
	}

	if repository.updatedUserID != testUser.ID() {
		t.Fatalf(
			"expected updated user ID %q, got %q",
			testUser.ID(),
			repository.updatedUserID,
		)
	}

	if repository.updatedFullName.String() != "Updated Name" {
		t.Fatalf(
			"expected updated full name %q, got %q",
			"Updated Name",
			repository.updatedFullName.String(),
		)
	}
}

func TestUpdateUserProfileUseCase_Execute_LogsSuccess(t *testing.T) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.succeeded",
		"update_user_profile",
		"",
	)

	if logger.event.UserID != testUser.ID().String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			testUser.ID().String(),
			logger.event.UserID,
		)
	}

	if logger.event.Role != user.RoleUser.String() {
		t.Fatalf(
			"expected role %q, got %q",
			user.RoleUser.String(),
			logger.event.Role,
		)
	}
}

func TestUpdateUserProfileUseCase_Execute_LogsInvalidUserID(
	t *testing.T,
) {
	repository := &fakeUpdateProfileRepository{}
	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		"invalid-user-id",
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
	)

	if err == nil {
		t.Fatal("expected invalid user ID error, got nil")
	}

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.validation_failed",
		"update_user_profile",
		"validation_failed",
	)
}

func TestUpdateUserProfileUseCase_Execute_LogsInvalidFullName(
	t *testing.T,
) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "",
		},
	)

	if err == nil {
		t.Fatal("expected invalid full name error, got nil")
	}

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.validation_failed",
		"update_user_profile",
		"validation_failed",
	)
}

func TestUpdateUserProfileUseCase_Execute_LogsUserLookupFailure(
	t *testing.T,
) {
	expectedErr := errors.New("database failure")

	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user:        testUser,
		findByIDErr: expectedErr,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
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

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.failed",
		"update_user_profile",
		"user_lookup_failed",
	)
}

func TestUpdateUserProfileUseCase_Execute_LogsUserNotFound(
	t *testing.T,
) {
	repository := &fakeUpdateProfileRepository{
		user: nil,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		user.NewUserID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
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

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.not_found",
		"update_user_profile",
		"user_not_found",
	)
}

func TestUpdateUserProfileUseCase_Execute_LogsPersistenceFailure(
	t *testing.T,
) {
	expectedErr := errors.New("database update failure")

	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user:              testUser,
		updateFullNameErr: expectedErr,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
	)

	if err == nil {
		t.Fatal("expected persistence error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected persistence error %v, got %v",
			expectedErr,
			err,
		)
	}

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.failed",
		"update_user_profile",
		"profile_update_failed",
	)
}

func TestUpdateUserProfileUseCase_Execute_LoggerFailureDoesNotAffectResult(
	t *testing.T,
) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	logger := &fakeUpdateProfileLogger{
		err: errors.New("logger failure"),
	}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected logger failure not to affect result, got %v",
			err,
		)
	}

	if result.FullName != "Updated Name" {
		t.Fatalf(
			"expected updated full name, got %q",
			result.FullName,
		)
	}

	assertUpdateProfileEvent(
		t,
		logger,
		"user.profile_update.succeeded",
		"update_user_profile",
		"",
	)
}

func TestUpdateUserProfileUseCase_Execute_DoesNotLogSensitiveAuthenticationData(
	t *testing.T,
) {
	testUser := newProfileUpdateUser(t)

	repository := &fakeUpdateProfileRepository{
		user: testUser,
	}

	logger := &fakeUpdateProfileLogger{}

	useCase := use_cases.NewUpdateUserProfileUseCase(
		repository,
		logger,
	)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserProfileCommand{
			FullName: "Updated Name",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if logger.event.Event != "user.profile_update.succeeded" {
		t.Fatalf(
			"expected success event, got %q",
			logger.event.Event,
		)
	}

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

var _ ports.UserRepository = (*fakeUpdateProfileRepository)(nil)
var _ ports.Logger = (*fakeUpdateProfileLogger)(nil)
