package schemas

// ActivationEmailRequest represents an activation email request.
type ActivationEmailRequest struct {
	Email string `json:"email"`
}

// VerifyEmailRequest represents an email verification request.
type VerifyEmailRequest struct {
	Token string `json:"token"`
}
