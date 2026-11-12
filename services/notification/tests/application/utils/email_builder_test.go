package utils_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/utils"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

func TestBuildEmail(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   []byte
		wantErr   error
	}{
		{
			name:      "unsupported event",
			eventType: "unknown.event",
			payload:   []byte(`{}`),
			wantErr:   application.ErrUnsupportedEmailEvent,
		},
		{
			name:    "empty event type",
			payload: []byte(`{}`),
			wantErr: application.ErrUnsupportedEmailEvent,
		},
		{
			name:      "invalid JSON",
			eventType: utils.IdentityActivationRequested,
			payload:   []byte(`{`),
		},
		{
			name:      "invalid recipient",
			eventType: utils.IdentityActivationRequested,
			payload:   []byte(`{"recipient":"invalid","activation_url":"http://localhost:3000/activate"}`),
		},
		{
			name:      "invalid activation URL",
			eventType: utils.IdentityActivationRequested,
			payload:   []byte(`{"recipient":"user@example.com","activation_url":"javascript:alert(1)"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := utils.BuildEmail(tt.eventType, tt.payload)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if email != (entities.Email{}) {
					t.Fatal("expected zero-value email on error")
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	t.Run("valid activation event", func(t *testing.T) {
		payload := []byte(`{"recipient":"user@example.com","activation_url":"http://localhost:3000/activate?token=abc"}`)

		email, err := utils.BuildEmail(utils.IdentityActivationRequested, payload)
		if err != nil {
			t.Fatal(err)
		}

		if email.Recipient().Value() != "user@example.com" {
			t.Fatalf("unexpected recipient: %s", email.Recipient().Value())
		}
		if email.Subject() != "Activate your account" {
			t.Fatalf("unexpected subject: %s", email.Subject())
		}
		if email.Format() != value_objects.EmailFormatHTML {
			t.Fatalf("unexpected format: %s", email.Format())
		}
		if !contains(email.Body(), "Activate account") || !contains(email.Body(), "token=abc") {
			t.Fatalf("activation body missing expected content: %s", email.Body())
		}
	})
}

func contains(value, substring string) bool {
	return len(substring) == 0 || len(value) >= len(substring) && func() bool {
		for i := 0; i+len(substring) <= len(value); i++ {
			if value[i:i+len(substring)] == substring {
				return true
			}
		}
		return false
	}()
}
