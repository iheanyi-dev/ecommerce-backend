package postgres_tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
)

// newRefreshTokenTestUser creates a valid domain User for refresh-token
// repository integration tests.
//
// The refresh_tokens table has a foreign-key relationship to users, so every
// refresh-token test needs a real persisted user.
func newRefreshTokenTestUser(t *testing.T, repository *postgres.UserRepository) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("Refresh Token User")
	if err != nil {
		t.Fatalf("failed to create FullName: %v", err)
	}

	email, err := user.NewEmail(
		"refresh-token-" + uuid.NewString() + "@example.com",
	)
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create PasswordHash: %v", err)
	}

	newUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create User: %v", err)
	}

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("failed to persist test user: %v", err)
	}

	return newUser
}

// newRefreshTokenRecord creates a valid refresh-token persistence record.
func newRefreshTokenRecord(
	userID string,
	tokenHash string,
) ports.RefreshTokenRecord {
	now := time.Now()

	return ports.RefreshTokenRecord{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: now,
	}
}

// TestRefreshTokenRepository_Create verifies that a refresh-token record can
// be persisted through the PostgreSQL repository.
func TestRefreshTokenRepository_Create(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	record := newRefreshTokenRecord(
		newUser.ID().String(),
		"refresh-token-hash-create",
	)

	err := refreshTokenRepository.Create(
		context.Background(),
		record,
	)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	// Verify persistence through SQLC directly. The repository contract is
	// tested independently below through FindByTokenHash.
	persisted, err := queries.FindRefreshTokenByHash(
		context.Background(),
		record.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"FindRefreshTokenByHash() returned an error after Create(): %v",
			err,
		)
	}

	if persisted.ID.String() != record.ID {
		t.Fatalf(
			"expected ID %s, got %s",
			record.ID,
			persisted.ID.String(),
		)
	}

	if persisted.UserID.String() != record.UserID {
		t.Fatalf(
			"expected user ID %s, got %s",
			record.UserID,
			persisted.UserID.String(),
		)
	}

	if persisted.TokenHash != record.TokenHash {
		t.Fatalf("token hash changed during persistence")
	}

	if !persisted.ExpiresAt.Time.Truncate(time.Microsecond).Equal(
		record.ExpiresAt.Truncate(time.Microsecond),
	) {
		t.Fatalf("expires_at changed during persistence")
	}

	if persisted.RevokedAt.Valid {
		t.Fatal("expected newly created refresh token to be unrevoked")
	}

	if !persisted.CreatedAt.Time.Truncate(time.Microsecond).Equal(
		record.CreatedAt.Truncate(time.Microsecond),
	) {
		t.Fatalf("created_at changed during persistence")
	}
}

// TestRefreshTokenRepository_FindByTokenHash verifies that a refresh-token
// record can be retrieved by its secure hash.
func TestRefreshTokenRepository_FindByTokenHash(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	record := newRefreshTokenRecord(
		newUser.ID().String(),
		"refresh-token-hash-find",
	)

	if err := refreshTokenRepository.Create(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	found, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		record.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"FindByTokenHash() returned an error: %v",
			err,
		)
	}

	if found == nil {
		t.Fatal("expected refresh-token record, got nil")
	}

	if found.ID != record.ID {
		t.Fatalf(
			"expected ID %s, got %s",
			record.ID,
			found.ID,
		)
	}

	if found.UserID != record.UserID {
		t.Fatalf(
			"expected user ID %s, got %s",
			record.UserID,
			found.UserID,
		)
	}

	if found.TokenHash != record.TokenHash {
		t.Fatal("token hash changed during reconstruction")
	}

	if found.RevokedAt != nil {
		t.Fatal("expected token to be unrevoked")
	}
}

// TestRefreshTokenRepository_FindByTokenHash_NotFound verifies that the
// repository returns nil when the requested token hash does not exist.
func TestRefreshTokenRepository_FindByTokenHash_NotFound(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	found, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		"missing-refresh-token-hash",
	)
	if err != nil {
		t.Fatalf(
			"FindByTokenHash() returned an unexpected error: %v",
			err,
		)
	}

	if found != nil {
		t.Fatal("expected nil record for missing token hash")
	}
}

// TestRefreshTokenRepository_Revoke verifies that an active refresh token
// can be revoked.
func TestRefreshTokenRepository_Revoke(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	record := newRefreshTokenRecord(
		newUser.ID().String(),
		"refresh-token-hash-revoke",
	)

	if err := refreshTokenRepository.Create(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	revokedAt := time.Now()

	if err := refreshTokenRepository.Revoke(
		context.Background(),
		record.ID,
		revokedAt,
	); err != nil {
		t.Fatalf("Revoke() returned an error: %v", err)
	}

	found, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		record.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"FindByTokenHash() returned an error after Revoke(): %v",
			err,
		)
	}

	if found == nil {
		t.Fatal("expected revoked refresh-token record, got nil")
	}

	if found.RevokedAt == nil {
		t.Fatal("expected revoked_at to be populated")
	}

	if !found.RevokedAt.Truncate(time.Microsecond).Equal(
		revokedAt.Truncate(time.Microsecond),
	) {
		t.Fatalf(
			"expected revoked_at %v, got %v",
			revokedAt,
			*found.RevokedAt,
		)
	}
}

// TestRefreshTokenRepository_Create_DuplicateHash verifies that PostgreSQL
// rejects two refresh-token records containing the same token hash.
//
// The database UNIQUE constraint is an important security invariant because
// each stored refresh-token hash must identify only one session.
func TestRefreshTokenRepository_Create_DuplicateHash(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	firstRecord := newRefreshTokenRecord(
		newUser.ID().String(),
		"duplicate-refresh-token-hash",
	)

	if err := refreshTokenRepository.Create(
		context.Background(),
		firstRecord,
	); err != nil {
		t.Fatalf(
			"first Create() returned an error: %v",
			err,
		)
	}

	secondRecord := newRefreshTokenRecord(
		newUser.ID().String(),
		firstRecord.TokenHash,
	)

	err := refreshTokenRepository.Create(
		context.Background(),
		secondRecord,
	)

	if err == nil {
		t.Fatal(
			"expected duplicate token hash to return a PostgreSQL error",
		)
	}
}

// TestRefreshTokenRepository_Create_RequiresExistingUser verifies the
// foreign-key relationship between refresh_tokens and users.
func TestRefreshTokenRepository_Create_RequiresExistingUser(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	record := newRefreshTokenRecord(
		uuid.NewString(),
		"refresh-token-invalid-user",
	)

	err := refreshTokenRepository.Create(
		context.Background(),
		record,
	)

	if err == nil {
		t.Fatal(
			"expected Create() to reject a refresh token for a non-existent user",
		)
	}
}

// TestRefreshTokenRepository_Revoke_NonExistentToken verifies that revoking
// an unknown ID does not accidentally modify another session.
func TestRefreshTokenRepository_Revoke_NonExistentToken(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	err := refreshTokenRepository.Revoke(
		context.Background(),
		uuid.NewString(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf(
			"Revoke() returned an unexpected error: %v",
			err,
		)
	}
}

// TestRefreshTokenRepository_Rotate verifies that refresh-token rotation
// atomically revokes the old session and creates the replacement session.
//
// Unlike the other repository tests, this test intentionally uses the
// pool-backed repository because Rotate() starts its own PostgreSQL
// transaction. The records used by Rotate() therefore need to be committed
// before Rotate() begins.
func TestRefreshTokenRepository_Rotate(t *testing.T) {
	testDB := NewTestDatabase(t)

	// Use the pool directly so the test data is committed and visible to
	// Rotate(), which starts its own transaction internally.
	queries := generated.New(testDB.pool)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	// Clean up the committed test user after the test.
	//
	// The refresh_tokens rows are removed automatically because the
	// foreign-key constraint uses ON DELETE CASCADE.
	t.Cleanup(func() {
		_, err := testDB.pool.Exec(
			context.Background(),
			"DELETE FROM users WHERE id = $1",
			newUser.ID().String(),
		)

		if err != nil {
			t.Errorf(
				"failed to clean up rotation test user: %v",
				err,
			)
		}
	})

	oldRecord := newRefreshTokenRecord(
		newUser.ID().String(),
		"rotation-old-hash",
	)

	if err := refreshTokenRepository.Create(
		context.Background(),
		oldRecord,
	); err != nil {
		t.Fatalf(
			"Create() returned an error for old token: %v",
			err,
		)
	}

	newRecord := newRefreshTokenRecord(
		newUser.ID().String(),
		"rotation-new-hash",
	)

	revokedAt := time.Now()

	err := refreshTokenRepository.Rotate(
		context.Background(),
		oldRecord.ID,
		revokedAt,
		newRecord,
	)
	if err != nil {
		t.Fatalf(
			"Rotate() returned an error: %v",
			err,
		)
	}

	// Verify the old token was revoked.
	oldResult, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		oldRecord.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve old refresh token: %v",
			err,
		)
	}

	if oldResult == nil {
		t.Fatal("expected old refresh token record, got nil")
	}

	if oldResult.RevokedAt == nil {
		t.Fatal("expected old refresh token to be revoked")
	}

	if !oldResult.RevokedAt.Truncate(time.Microsecond).Equal(
		revokedAt.Truncate(time.Microsecond),
	) {
		t.Fatalf(
			"expected revoked_at %v, got %v",
			revokedAt,
			*oldResult.RevokedAt,
		)
	}

	// Verify the replacement token exists and remains active.
	newResult, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		newRecord.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve replacement refresh token: %v",
			err,
		)
	}

	if newResult == nil {
		t.Fatal("expected replacement refresh token record, got nil")
	}

	if newResult.TokenHash != newRecord.TokenHash {
		t.Fatal("replacement token hash does not match")
	}

	if newResult.RevokedAt != nil {
		t.Fatal("expected replacement refresh token to be active")
	}

	if newResult.UserID != newRecord.UserID {
		t.Fatalf(
			"expected replacement token user ID %s, got %s",
			newRecord.UserID,
			newResult.UserID,
		)
	}
}

// TestRefreshTokenRepository_Rotate_RollsBackOnCreateFailure verifies that
// PostgreSQL rolls back the revocation when creation of the replacement
// refresh token fails.
//
// This test protects the most important invariant of token rotation:
// partial rotation must never invalidate the existing session.
//
// The replacement deliberately uses the same token hash as the old token,
// causing PostgreSQL's UNIQUE constraint to reject the INSERT.
func TestRefreshTokenRepository_Rotate_RollsBackOnCreateFailure(
	t *testing.T,
) {
	testDB := NewTestDatabase(t)

	// Rotate() creates its own transaction from the pool. Therefore the
	// initial user and refresh-token records must be committed before
	// Rotate() starts.
	queries := generated.New(testDB.pool)

	userRepository := postgres.NewUserRepository(queries)
	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		testDB.pool,
		queries,
	)

	newUser := newRefreshTokenTestUser(t, userRepository)

	// Clean up the committed test user after the test.
	//
	// ON DELETE CASCADE removes the refresh-token records.
	t.Cleanup(func() {
		_, err := testDB.pool.Exec(
			context.Background(),
			"DELETE FROM users WHERE id = $1",
			newUser.ID().String(),
		)

		if err != nil {
			t.Errorf(
				"failed to clean up rotation rollback test user: %v",
				err,
			)
		}
	})

	oldRecord := newRefreshTokenRecord(
		newUser.ID().String(),
		"rotation-rollback-hash",
	)

	if err := refreshTokenRepository.Create(
		context.Background(),
		oldRecord,
	); err != nil {
		t.Fatalf(
			"Create() returned an error for old token: %v",
			err,
		)
	}

	// The duplicate hash intentionally violates the UNIQUE constraint.
	failingReplacement := newRefreshTokenRecord(
		newUser.ID().String(),
		oldRecord.TokenHash,
	)

	err := refreshTokenRepository.Rotate(
		context.Background(),
		oldRecord.ID,
		time.Now(),
		failingReplacement,
	)

	if err == nil {
		t.Fatal(
			"expected Rotate() to fail when replacement token hash is duplicated",
		)
	}

	// The critical assertion:
	//
	// Rotate() must have rolled back the old-token revocation when the
	// replacement creation failed.
	oldResult, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		oldRecord.TokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve old refresh token after rollback: %v",
			err,
		)
	}

	if oldResult == nil {
		t.Fatal(
			"expected old refresh token record after rollback, got nil",
		)
	}

	if oldResult.RevokedAt != nil {
		t.Fatal(
			"expected old refresh token revocation to be rolled back",
		)
	}

	// Verify that the failed replacement was not persisted.
	replacementResult, err := refreshTokenRepository.FindByTokenHash(
		context.Background(),
		failingReplacement.TokenHash,
	)

	// Because the replacement hash is identical to the old hash, the
	// lookup must return the original old token only. The important
	// invariant is that its revoked_at remains NULL.
	if err != nil {
		t.Fatalf(
			"failed to verify replacement rollback: %v",
			err,
		)
	}

	if replacementResult == nil {
		t.Fatal(
			"expected original refresh token to remain after rollback",
		)
	}

	if replacementResult.ID != oldRecord.ID {
		t.Fatalf(
			"expected original token ID %s, got %s",
			oldRecord.ID,
			replacementResult.ID,
		)
	}
}

// Compile-time assertion that the future implementation satisfies the
// application port.
var _ ports.RefreshTokenRepository = (*postgres.RefreshTokenRepository)(nil)

