package utils

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

const IdentityActivationRequested = "identity.activation_requested"

type activationRequestedPayload struct {
	Recipient     string `json:"recipient"`
	ActivationURL string `json:"activation_url"`
}

func BuildEmail(eventType string, payload []byte) (entities.Email, error) {
	switch eventType {
	case IdentityActivationRequested:
		return buildActivationEmail(payload)
	default:
		return entities.Email{}, application.ErrUnsupportedEmailEvent
	}
}

func buildActivationEmail(payload []byte) (entities.Email, error) {
	var event activationRequestedPayload

	if err := json.Unmarshal(payload, &event); err != nil {
		return entities.Email{}, fmt.Errorf("decode activation email event: %w", err)
	}

	recipient, err := value_objects.NewEmailAddress(event.Recipient)
	if err != nil {
		return entities.Email{}, err
	}

	activationURL := strings.TrimSpace(event.ActivationURL)
	parsedURL, err := url.ParseRequestURI(activationURL)
	if err != nil || parsedURL.Scheme != "http" && parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return entities.Email{}, fmt.Errorf("invalid activation URL")
	}

	textBody := "Welcome!\n\nPlease activate your account by visiting the link below:\n\n" +
		activationURL + "\n\nIf you did not create this account, you can ignore this email."

	htmlBody := "<h2>Welcome!</h2>" +
		"<p>Please activate your account by clicking the link below:</p>" +
		`<p><a href="` + html.EscapeString(activationURL) + `">Activate account</a></p>` +
		"<p>If you did not create this account, you can ignore this email.</p>"

	_ = textBody
	return entities.NewEmail(
		recipient,
		"Activate your account",
		htmlBody,
		value_objects.EmailFormatHTML,
	)
}
