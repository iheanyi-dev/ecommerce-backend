package entities_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

func TestNewEmail(t *testing.T) {
	recipient, err := value_objects.NewEmailAddress("customer@example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		recipient value_objects.EmailAddress
		subject   string
		body      string
		format    value_objects.EmailFormat
		wantErr   error
	}{
		{name: "plain text email", recipient: recipient, subject: "Order confirmation", body: "Your order has been confirmed.", format: value_objects.EmailFormatPlainText},
		{name: "HTML email", recipient: recipient, subject: "Order shipped", body: "<h1>Your order has shipped.</h1>", format: value_objects.EmailFormatHTML},
		{name: "empty subject", recipient: recipient, subject: " ", body: "Email body", format: value_objects.EmailFormatPlainText, wantErr: domain.ErrEmptyEmailSubject},
		{name: "empty body", recipient: recipient, subject: "Email subject", body: "\t", format: value_objects.EmailFormatPlainText, wantErr: domain.ErrEmptyEmailBody},
		{name: "unsupported format", recipient: recipient, subject: "Email subject", body: "Email body", format: "application/pdf", wantErr: domain.ErrInvalidEmailFormat},
		{name: "zero value recipient", subject: "Email subject", body: "Email body", format: value_objects.EmailFormatPlainText, wantErr: domain.ErrInvalidEmailAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entities.NewEmail(tt.recipient, tt.subject, tt.body, tt.format)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if err != nil {
				return
			}

			if got.Recipient().Value() != recipient.Value() {
				t.Errorf("unexpected recipient: %q", got.Recipient().Value())
			}
			if got.Subject() != tt.subject {
				t.Errorf("expected subject %q, got %q", tt.subject, got.Subject())
			}
			if got.Body() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, got.Body())
			}
			if got.Format() != tt.format {
				t.Errorf("expected format %q, got %q", tt.format, got.Format())
			}
		})
	}
}
