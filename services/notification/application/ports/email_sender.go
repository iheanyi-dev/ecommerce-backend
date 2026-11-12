package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
)

type EmailSender interface {
	Send(ctx context.Context, email entities.Email) error
}
