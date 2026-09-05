package dto

// RefreshTokenCommand contains the refresh token supplied by a client
// requesting a new authentication session.
//
// The raw refresh token exists only at the application boundary and should
// never be persisted directly.
type RefreshTokenCommand struct {
	RefreshToken string
}

// RefreshTokenResult contains the newly issued authentication credentials.
//
// The old refresh token is never returned. Refresh-token rotation produces
// a new refresh token for every successful refresh operation.
type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
}
