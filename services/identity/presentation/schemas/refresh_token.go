package schemas

// RefreshTokenRequest represents the JSON payload submitted when
// a client requests a new access token using a refresh token.
//
// The refresh token is intentionally treated as an opaque string at
// the presentation boundary. Token validation and rotation belong to
// the application layer.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents the successful refresh response.
//
// A successful refresh rotates the previous refresh token and returns
// both a newly issued access token and its replacement refresh token.
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}