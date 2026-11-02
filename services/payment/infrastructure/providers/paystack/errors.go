package paystack

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfiguration = errors.New("invalid Paystack client configuration")
	ErrInvalidRequestPath   = errors.New("invalid Paystack request path")
	ErrResponseTooLarge     = errors.New("Paystack response exceeds size limit")
	ErrInvalidResponse      = errors.New("invalid Paystack response")
	ErrAPIRequestFailed     = errors.New("Paystack API request failed")
)

// ProviderError represents a rejected request or unsuccessful API response.
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Paystack API request failed with HTTP status %d", e.StatusCode)
	}
	return fmt.Sprintf("Paystack API request failed with HTTP status %d: %s", e.StatusCode, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return ErrAPIRequestFailed
}
