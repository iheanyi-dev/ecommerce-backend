package schemas

// LogoutRequest represents the JSON payload submitted when
// a user wants to log out of a specific authenticated session.
//
// The refresh token identifies the exact device/session that
// should be revoked. Other sessions remain active.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
