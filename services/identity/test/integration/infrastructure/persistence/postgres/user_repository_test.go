package postgres_tests

import (
	"context"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
)

// newTestUser creates a valid domain User for repository tests.
//
// The repository receives domain aggregates, so the tests should construct
// the User through the domain API rather than constructing persistence
// models directly.
func newTestUser(t *testing.T, email string) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create FullName: %v", err)
	}

	userEmail, err := user.NewEmail(email)
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create PasswordHash: %v", err)
	}

	newUser, err := user.NewUser(
		fullName,
		userEmail,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create User: %v", err)
	}

	return newUser
}

// TestUserRepository_Create verifies that a valid domain User can be
// persisted through the PostgreSQL repository.
func TestUserRepository_Create(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	// SQLC generates a query executor backed by the transaction.
	queries := generated.New(tx)

	// The repository depends on the SQLC-generated query executor rather
	// than depending directly on PostgreSQL.
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "create@example.com")

	err := repository.Create(
		context.Background(),
		newUser,
	)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	// We verify persistence through the repository's public contract.
	exists, err := repository.ExistsByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf(
			"ExistsByEmail() returned an error after Create(): %v",
			err,
		)
	}

	if !exists {
		t.Fatal("expected created user to exist")
	}
}

// TestUserRepository_ExistsByEmail_UserDoesNotExist verifies that the
// repository returns false when no user has the requested email.
func TestUserRepository_ExistsByEmail_UserDoesNotExist(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	email, err := user.NewEmail("missing@example.com")
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	exists, err := repository.ExistsByEmail(
		context.Background(),
		email,
	)
	if err != nil {
		t.Fatalf("ExistsByEmail() returned an error: %v", err)
	}

	if exists {
		t.Fatal("expected non-existent user to return false")
	}
}

// TestUserRepository_ExistsByEmail_UserExists verifies that the repository
// returns true when a user with the requested email exists.
func TestUserRepository_ExistsByEmail_UserExists(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "exists@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	exists, err := repository.ExistsByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf("ExistsByEmail() returned an error: %v", err)
	}

	if !exists {
		t.Fatal("expected existing user to return true")
	}
}

// TestUserRepository_FindByEmail_UserExists verifies that the repository
// retrieves an existing user and correctly reconstructs the domain
// aggregate from the raw values returned by PostgreSQL.
func TestUserRepository_FindByEmail_UserExists(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "find@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	reconstitutedUser, err := repository.FindByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf("FindByEmail() returned an error: %v", err)
	}

	if reconstitutedUser == nil {
		t.Fatal("expected user, got nil")
	}

	// Verify that the repository preserved the complete domain identity.
	if reconstitutedUser.ID() != newUser.ID() {
		t.Fatalf("expected ID %s, got %s",
			newUser.ID().String(),
			reconstitutedUser.ID().String(),
		)
	}

	// Verify that the persisted FullName was reconstructed into the
	// domain FullName value object correctly.
	if reconstitutedUser.FullName() != newUser.FullName() {
		t.Fatalf("full name changed during reconstruction")
	}

	// Verify that the persisted Email was reconstructed into the
	// domain Email value object correctly.
	if reconstitutedUser.Email() != newUser.Email() {
		t.Fatalf("email changed during reconstruction")
	}

	// Verify that the persisted PasswordHash was reconstructed without
	// modifying the stored hash.
	if reconstitutedUser.PasswordHash() != newUser.PasswordHash() {
		t.Fatalf("password hash changed during reconstruction")
	}

	// Verify that role and status were reconstructed from their
	// persisted string representations.
	if reconstitutedUser.Role() != newUser.Role() {
		t.Fatalf("role changed during reconstruction")
	}

	if reconstitutedUser.Status() != newUser.Status() {
		t.Fatalf("status changed during reconstruction")
	}

	// PostgreSQL TIMESTAMPTZ stores timestamps with microsecond precision.
	// time.Now() may contain nanoseconds, so compare at the same precision
	// used by PostgreSQL.
	if !reconstitutedUser.CreatedAt().Truncate(time.Microsecond).Equal(
		newUser.CreatedAt().Truncate(time.Microsecond),
	) {
		t.Fatalf("created_at changed during reconstruction")
	}

	if !reconstitutedUser.UpdatedAt().Truncate(time.Microsecond).Equal(
		newUser.UpdatedAt().Truncate(time.Microsecond),
	) {
		t.Fatalf("updated_at changed during reconstruction")
	}
}

// TestUserRepository_FindByEmail_UserDoesNotExist verifies that the
// repository returns the appropriate error when no user exists with
// the requested email.
func TestUserRepository_FindByEmail_UserDoesNotExist(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	email, err := user.NewEmail("missing-find@example.com")
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	_, err = repository.FindByEmail(
		context.Background(),
		email,
	)

	if err == nil {
		t.Fatal("expected FindByEmail() to return an error")
	}
}

// TestUserRepository_FindByID_UserExists verifies that the repository
// retrieves an existing user by its domain UserID and correctly
// reconstructs the complete domain aggregate.
func TestUserRepository_FindByID_UserExists(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "find-by-id@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	reconstitutedUser, err := repository.FindByID(
		context.Background(),
		newUser.ID(),
	)
	if err != nil {
		t.Fatalf("FindByID() returned an error: %v", err)
	}

	if reconstitutedUser == nil {
		t.Fatal("expected user, got nil")
	}

	if reconstitutedUser.ID() != newUser.ID() {
		t.Fatalf(
			"expected ID %s, got %s",
			newUser.ID().String(),
			reconstitutedUser.ID().String(),
		)
	}

	if reconstitutedUser.FullName() != newUser.FullName() {
		t.Fatalf("full name changed during reconstruction")
	}

	if reconstitutedUser.Email() != newUser.Email() {
		t.Fatalf("email changed during reconstruction")
	}

	if reconstitutedUser.PasswordHash() != newUser.PasswordHash() {
		t.Fatalf("password hash changed during reconstruction")
	}

	if reconstitutedUser.Role() != newUser.Role() {
		t.Fatalf("role changed during reconstruction")
	}

	if reconstitutedUser.Status() != newUser.Status() {
		t.Fatalf("status changed during reconstruction")
	}
}

// TestUserRepository_FindByID_UserDoesNotExist verifies that the repository
// returns an error when no user exists with the requested UserID.
func TestUserRepository_FindByID_UserDoesNotExist(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	nonExistentID := user.NewUserID()

	_, err := repository.FindByID(
		context.Background(),
		nonExistentID,
	)

	if err == nil {
		t.Fatal("expected FindByID() to return an error")
	}
}

// TestUserRepository_UpdateFullName verifies that the repository can persist
// a user's changed full name without changing immutable account information.
func TestUserRepository_UpdateFullName(t *testing.T) {
	t.Helper()

	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	// SQLC uses the transaction as the query executor.
	queries := generated.New(tx)

	// The repository translates between the domain and PostgreSQL layers.
	repository := postgres.NewUserRepository(queries)

	// Create the initial persisted user using the existing test helper.
	testUser := newTestUser(t, "update-full-name@example.com")

	if err := repository.Create(
		context.Background(),
		testUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	// Construct the new name through the domain value object so the test
	// exercises the same validation boundary used by the application.
	newFullName, err := user.NewFullName("Updated Test User")
	if err != nil {
		t.Fatalf("failed to create updated FullName: %v", err)
	}

	testUser.ChangeFullName(newFullName)

	// Update only the mutable profile field.
	if err := repository.UpdateFullName(
		context.Background(),
		testUser,
	); err != nil {
		t.Fatalf("UpdateFullName() returned an error: %v", err)
	}

	// Read the aggregate back from PostgreSQL to verify actual persistence.
	updatedUser, err := repository.FindByID(
		context.Background(),
		testUser.ID(),
	)
	if err != nil {
		t.Fatalf("FindByID() returned an error: %v", err)
	}

	if updatedUser == nil {
		t.Fatal("expected updated user, got nil")
	}

	if updatedUser.FullName().String() != "Updated Test User" {
		t.Fatalf(
			"expected full name %q, got %q",
			"Updated Test User",
			updatedUser.FullName().String(),
		)
	}

	// Email is immutable in Phase 7 and must remain unchanged.
	if updatedUser.Email() != testUser.Email() {
		t.Fatal("expected email to remain unchanged")
	}
}

// TestUserRepository_UpdatePasswordHash verifies that an existing user's
// password hash can be updated through the repository and subsequently
// reconstructed correctly from PostgreSQL.
func TestUserRepository_UpdatePasswordHash(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	// Create the user with the original password hash first.
	newUser := newTestUser(t, "update-password@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	// Create the new password hash that should replace the existing hash.
	newPasswordHash, err := user.NewPasswordHash(
		"hashed-new-password",
	)
	if err != nil {
		t.Fatalf("failed to create new PasswordHash: %v", err)
	}
	newUser.ChangePassword(newPasswordHash)

	// Act: update only the persisted password hash.
	err = repository.UpdatePasswordHash(
		context.Background(),
		newUser,
	)
	if err != nil {
		t.Fatalf(
			"UpdatePasswordHash() returned an error: %v",
			err,
		)
	}

	// Verify the update through the repository's public contract.
	updatedUser, err := repository.FindByID(
		context.Background(),
		newUser.ID(),
	)
	if err != nil {
		t.Fatalf(
			"FindByID() returned an error after UpdatePasswordHash(): %v",
			err,
		)
	}

	if updatedUser == nil {
		t.Fatal("expected updated user, got nil")
	}

	// The user's identity must remain unchanged.
	if updatedUser.ID() != newUser.ID() {
		t.Fatalf(
			"expected ID %s, got %s",
			newUser.ID().String(),
			updatedUser.ID().String(),
		)
	}

	// The repository must persist the new password hash exactly as
	// supplied by the application layer.
	if updatedUser.PasswordHash() != newPasswordHash {
		t.Fatalf(
			"expected password hash %q, got %q",
			newPasswordHash.String(),
			updatedUser.PasswordHash().String(),
		)
	}

	// Other user properties must remain unchanged by a password update.
	if updatedUser.FullName() != newUser.FullName() {
		t.Fatal("full name changed during password update")
	}

	if updatedUser.Email() != newUser.Email() {
		t.Fatal("email changed during password update")
	}

	if updatedUser.Role() != newUser.Role() {
		t.Fatal("role changed during password update")
	}

	if updatedUser.Status() != newUser.Status() {
		t.Fatal("status changed during password update")
	}
}
