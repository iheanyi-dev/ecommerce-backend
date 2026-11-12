package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
)

type SendEmailUseCase struct {
	emailSender ports.EmailSender
}

func NewSendEmailUseCase(
	emailSender ports.EmailSender,
) (*SendEmailUseCase, error) {
	if emailSender == nil {
		return nil, application.ErrEmailSenderRequired
	}

	return &SendEmailUseCase{
		emailSender: emailSender,
	}, nil
}

func (uc *SendEmailUseCase) Execute(
	ctx context.Context,
	email entities.Email,
) error {
	return uc.emailSender.Send(ctx, email)
}
