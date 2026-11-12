package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

type mockEmailSender struct {
	called bool
	email  entities.Email
	err    error
}

func (m *mockEmailSender) Send(
	_ context.Context,
	email entities.Email,
) error {
	m.called = true
	m.email = email
	return m.err
}

func TestNewSendEmailUseCase(t *testing.T) {
	uc, err := use_cases.NewSendEmailUseCase(nil)

	if uc != nil {
		t.Fatal("expected nil use case")
	}

	if !errors.Is(err, application.ErrEmailSenderRequired) {
		t.Fatalf("expected ErrEmailSenderRequired, got %v", err)
	}
}

func TestSendEmailUseCaseExecute(t *testing.T) {
	recipient, err := value_objects.NewEmailAddress("customer@example.com")
	if err != nil {
		t.Fatal(err)
	}

	email, err := entities.NewEmail(
		recipient,
		"Order confirmation",
		"Your order has been confirmed.",
		value_objects.EmailFormatPlainText,
	)
	if err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("email delivery failed")

	tests := []struct {
		name    string
		sendErr error
		wantErr error
	}{
		{
			name: "successful delivery",
		},
		{
			name:    "delivery failure",
			sendErr: expectedErr,
			wantErr: expectedErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockEmailSender{
				err: tt.sendErr,
			}

			uc, err := use_cases.NewSendEmailUseCase(sender)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			err = uc.Execute(ctx, email)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if !sender.called {
				t.Fatal("expected email sender to be called")
			}

			if sender.email.Recipient().Value() != email.Recipient().Value() {
				t.Error("unexpected recipient")
			}

			if sender.email.Subject() != email.Subject() {
				t.Error("unexpected subject")
			}

			if sender.email.Body() != email.Body() {
				t.Error("unexpected body")
			}

			if sender.email.Format() != email.Format() {
				t.Error("unexpected format")
			}
		})
	}
}
