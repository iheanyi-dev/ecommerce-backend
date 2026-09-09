package errors

import "errors"

// Authentication errors.
// ErrInvalidAccessToken indicates that an access token could not be
// validated successfully, including invalid, expired, or otherwise
// unacceptable access tokens.
var ErrInvalidAccessToken = errors.New("invalid access token")

// ErrInvalidCredentials indicates that the supplied authentication
// credentials could not be validated.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrAccountNotActive indicates that the user's account is not active
// and therefore cannot authenticate.
var ErrAccountNotActive = errors.New("account is not active")

// ErrTokenGeneration indicates that an access token could not be generated.
var ErrTokenGeneration = errors.New("failed to generate access token")

// Registration and user lookup errors.

// ErrEmailAlreadyExists indicates that the requested email address
// is already associated with another user.
var ErrEmailAlreadyExists = errors.New("email already exists")

// ErrUserNotFound indicates that the requested user does not exist.
var ErrUserNotFound = errors.New("user not found")

// Refresh-token errors.

// ErrInvalidRefreshToken indicates that the supplied refresh token
// is invalid or cannot be used.
var ErrInvalidRefreshToken = errors.New("invalid refresh token")

// ErrRefreshTokenGeneration indicates that a refresh token could not
// be generated.
var ErrRefreshTokenGeneration = errors.New("failed to generate refresh token")

// ErrRefreshTokenHashing indicates that a refresh token could not
// be securely hashed.
var ErrRefreshTokenHashing = errors.New("failed to hash refresh token")

// ErrRefreshTokenPersistence indicates that a generated refresh token
// could not be persisted.
var ErrRefreshTokenPersistence = errors.New("failed to persist refresh token")

// ErrRefreshTokenRevocation indicates that a refresh token could not
// be revoked.
var ErrRefreshTokenRevocation = errors.New("failed to revoke refresh token")

// Password policy errors.

// ErrPasswordTooShort indicates that the password does not meet the
// minimum length requirement.
var ErrPasswordTooShort = errors.New(
	"password must contain at least 8 characters",
)

// ErrPasswordTooLong indicates that the password exceeds the
// maximum permitted length.
var ErrPasswordTooLong = errors.New(
	"password must not exceed 64 characters",
)

// ErrPasswordMissingUppercase indicates that the password contains
// no uppercase character.
var ErrPasswordMissingUppercase = errors.New(
	"password must contain at least one uppercase letter",
)

// ErrPasswordMissingLowercase indicates that the password contains
// no lowercase character.
var ErrPasswordMissingLowercase = errors.New(
	"password must contain at least one lowercase letter",
)

// ErrPasswordMissingNumber indicates that the password contains
// no numeric character.
var ErrPasswordMissingNumber = errors.New(
	"password must contain at least one number",
)

// ErrPasswordMissingSpecialCharacter indicates that the password
// contains no special character.
var ErrPasswordMissingSpecialCharacter = errors.New(
	"password must contain at least one special character",
)
