package use_cases

import "errors"

// ErrInvalidCredentials is intentionally generic.
//
// Authentication must not reveal whether an email exists or whether
// the supplied password was incorrect. Both cases produce the same
// application-level error.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrAccountNotActive indicates that the credentials may be valid but
// the account is not currently permitted to authenticate.
//
// This error will eventually be mapped carefully at the presentation
// boundary so that sensitive account information is not unnecessarily
// exposed.
var ErrAccountNotActive = errors.New("account is not active")

// ErrTokenGeneration indicates that authentication succeeded but the
// access token could not be generated.
var ErrTokenGeneration = errors.New("failed to generate access token")
