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

type fakeUpdateUserStatusRepository struct {
	user *user.User

	updateStatusCalled bool
	updatedUserID      user.UserID
	updatedStatus      user.Status
	updatedAt          time.Time
}

func (f *fakeUpdateUserStatusRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	f.updateStatusCalled = true
	f.updatedUserID = existingUser.ID()
	f.updatedStatus = existingUser.Status()
	f.updatedAt = existingUser.UpdatedAt()

	return nil
}

func (f *fakeUpdateUserStatusRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakeUpdateUserStatusRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateUserStatusRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeUpdateUserStatusRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	return f.user, nil
}

func (f *fakeUpdateUserStatusRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateUserStatusRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeUpdateUserStatusRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
}

func newStatusUpdateUser(t *testing.T, status user.Status) *user.User {
	t.Helper()

	id := user.NewUserID()

	fullName, err := user.NewFullName("Status Test User")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("status@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	now := time.Now().UTC()

	return user.ReconstituteUser(
		id,
		fullName,
		email,
		passwordHash,
		user.RoleUser,
		status,
		now,
		now,
	)
}

func TestUpdateUserStatusUseCase_SuspendsActiveUser(t *testing.T) {
	testUser := newStatusUpdateUser(t, user.StatusActive)

	repository := &fakeUpdateUserStatusRepository{user: testUser}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	before := testUser.UpdatedAt()

	result, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserStatusCommand{
			Status: string(user.StatusSuspended),
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Status != string(user.StatusSuspended) {
		t.Fatalf("expected status %q, got %q",
			user.StatusSuspended,
			result.Status,
		)
	}

	if !repository.updateStatusCalled {
		t.Fatal("expected UpdateStatus to be called")
	}

	if repository.updatedUserID != testUser.ID() {
		t.Fatalf("expected updated user ID %q, got %q",
			testUser.ID(),
			repository.updatedUserID,
		)
	}

	if repository.updatedStatus != user.StatusSuspended {
		t.Fatalf("expected persisted status %q, got %q",
			user.StatusSuspended,
			repository.updatedStatus,
		)
	}

	if !repository.updatedAt.After(before) {
		t.Fatal("expected domain to update UpdatedAt")
	}
}

func TestUpdateUserStatusUseCase_ActivatesInactiveUser(t *testing.T) {
	testUser := newStatusUpdateUser(t, user.StatusInactive)

	repository := &fakeUpdateUserStatusRepository{user: testUser}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserStatusCommand{
			Status: string(user.StatusActive),
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if testUser.Status() != user.StatusActive {
		t.Fatalf("expected status %q, got %q",
			user.StatusActive,
			testUser.Status(),
		)
	}
}

func TestUpdateUserStatusUseCase_DeactivatesActiveUser(t *testing.T) {
	testUser := newStatusUpdateUser(t, user.StatusActive)

	repository := &fakeUpdateUserStatusRepository{user: testUser}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserStatusCommand{
			Status: string(user.StatusInactive),
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if testUser.Status() != user.StatusInactive {
		t.Fatalf("expected status %q, got %q",
			user.StatusInactive,
			testUser.Status(),
		)
	}
}

func TestUpdateUserStatusUseCase_RejectsInvalidStatus(t *testing.T) {
	testUser := newStatusUpdateUser(t, user.StatusActive)

	repository := &fakeUpdateUserStatusRepository{user: testUser}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserStatusCommand{
			Status: "deleted",
		},
	)
	if err == nil {
		t.Fatal("expected invalid status error")
	}

	if repository.updateStatusCalled {
		t.Fatal("expected UpdateStatus not to be called")
	}
}

func TestUpdateUserStatusUseCase_RejectsInvalidTransition(t *testing.T) {
	testUser := newStatusUpdateUser(t, user.StatusSuspended)

	repository := &fakeUpdateUserStatusRepository{user: testUser}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		testUser.ID().String(),
		dto.UpdateUserStatusCommand{
			Status: string(user.StatusActive),
		},
	)
	if err == nil {
		t.Fatal("expected invalid status transition error")
	}

	if repository.updateStatusCalled {
		t.Fatal("expected UpdateStatus not to be called")
	}
}

func TestUpdateUserStatusUseCase_ReturnsUserNotFound(t *testing.T) {
	repository := &fakeUpdateUserStatusRepository{}
	useCase := use_cases.NewUpdateUserStatusUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		user.NewUserID().String(),
		dto.UpdateUserStatusCommand{
			Status: string(user.StatusActive),
		},
	)
	if err != use_cases.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

var _ ports.UserRepository = (*fakeUpdateUserStatusRepository)(nil)
