package value_objects_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

func TestNewEmailAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "valid email", input: "customer@example.com", want: "customer@example.com"},
		{name: "surrounding whitespace", input: " customer@example.com ", want: "customer@example.com"},
		{name: "empty email", input: "", wantErr: domain.ErrInvalidEmailAddress},
		{name: "invalid email", input: "invalid-email", wantErr: domain.ErrInvalidEmailAddress},
		{name: "display name", input: "Customer <customer@example.com>", wantErr: domain.ErrInvalidEmailAddress},
		{name: "internal whitespace", input: "customer @example.com", wantErr: domain.ErrInvalidEmailAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := value_objects.NewEmailAddress(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if err != nil {
				return
			}

			if got.Value() != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got.Value())
			}
		})
	}
}
