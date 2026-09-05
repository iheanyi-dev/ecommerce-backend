package use_cases

import "errors"

var (
	// ErrInvalidRefreshToken indicates that the supplied refresh token
	// cannot be used to establish a new authentication session.
	//
	// This intentionally remains generic so the application does not
	// reveal whether the token was missing, expired, revoked, or unknown.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrRefreshTokenGeneration indicates that a new refresh token could
	// not be generated.
	ErrRefreshTokenGeneration = errors.New(
		"failed to generate refresh token",
	)

	// ErrRefreshTokenHashing indicates that a refresh token could not be
	// converted into its secure persistent representation.
	ErrRefreshTokenHashing = errors.New(
		"failed to hash refresh token",
	)

	// ErrRefreshTokenPersistence indicates that refresh-token session
	// state could not be persisted.
	ErrRefreshTokenPersistence = errors.New(
		"failed to persist refresh token",
	)

	// ErrRefreshTokenRevocation indicates that the previous refresh-token
	// session could not be revoked during rotation.
	ErrRefreshTokenRevocation = errors.New(
		"failed to revoke refresh token",
	)
)
