package value_objects_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

func TestNewEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    value_objects.EmailFormat
		wantErr error
	}{
		{name: "plain text", input: "text/plain", want: value_objects.EmailFormatPlainText},
		{name: "HTML", input: "text/html", want: value_objects.EmailFormatHTML},
		{name: "unsupported format", input: "application/pdf", wantErr: domain.ErrInvalidEmailFormat},
		{name: "empty format", input: "", wantErr: domain.ErrInvalidEmailFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := value_objects.NewEmailFormat(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if err != nil {
				return
			}

			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}

			if got.Value() != tt.input {
				t.Errorf("expected value %q, got %q", tt.input, got.Value())
			}
		})
	}
}
