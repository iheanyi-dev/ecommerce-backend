package email_test

import (
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/infrastructure/email"
)

func TestNewSMTPSender(t *testing.T) {
	tests := []struct {
		name    string
		config  *notification.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &notification.Config{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPFrom: "notifications@example.com",
			},
		},
		{name: "nil config", wantErr: true},
		{
			name: "missing host",
			config: &notification.Config{
				SMTPPort: 587, SMTPFrom: "notifications@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &notification.Config{
				SMTPHost: "smtp.example.com", SMTPPort: 0,
				SMTPFrom: "notifications@example.com",
			},
			wantErr: true,
		},
		{
			name: "missing sender",
			config: &notification.Config{
				SMTPHost: "smtp.example.com", SMTPPort: 587,
			},
			wantErr: true,
		},
		{
			name: "incomplete credentials",
			config: &notification.Config{
				SMTPHost: "smtp.example.com", SMTPPort: 587,
				SMTPFrom: "notifications@example.com", SMTPUsername: "username",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := email.NewSMTPSender(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSMTPSender() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	recipient, err := value_objects.NewEmailAddress("customer@example.com")
	if err != nil {
		t.Fatal(err)
	}

	format, err := value_objects.NewEmailFormat("text/html")
	if err != nil {
		t.Fatal(err)
	}

	message, err := entities.NewEmail(
		recipient,
		"Order shipped",
		"<h1>Your order has shipped</h1>",
		format,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := string(email.BuildMessageForTest(
		"notifications@example.com",
		"customer@example.com",
		"Order shipped",
		message,
	))

	for _, expected := range []string{
		"From: notifications@example.com\r\n",
		"To: customer@example.com\r\n",
		"Subject: Order shipped\r\n",
		"Content-Type: text/html; charset=UTF-8\r\n",
		"<h1>Your order has shipped</h1>",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("message does not contain %q", expected)
		}
	}
}
