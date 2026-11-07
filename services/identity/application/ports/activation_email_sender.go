package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// ActivationEmailSender delivers an account-activation link to a user.
type ActivationEmailSender interface {
	SendActivationEmail(
		ctx context.Context,
		email user.Email,
		activationURL string,
	) error
}
