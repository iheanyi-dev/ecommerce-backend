package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL       = "https://api.paystack.co"
	defaultTimeout       = 30 * time.Second
	maxResponseBodyBytes = 1 << 20 // 1 MiB
)

// Client is a small HTTP client for the Paystack REST API.
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Paystack client.
// An empty baseURL selects Paystack's production API endpoint.
// Supplying a custom baseURL allows isolated HTTP tests.
func NewClient(secretKey, baseURL string, httpClient *http.Client) (*Client, error) {
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		return nil, ErrInvalidConfiguration
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return nil, ErrInvalidConfiguration
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		secretKey:  secretKey,
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

type apiResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Request sends an authenticated request to a relative Paystack API endpoint
// and decodes the response's data field into result when result is non-nil.
func (c *Client) Request(
	ctx context.Context,
	method string,
	path string,
	payload any,
	result any,
) error {
	if c == nil || c.httpClient == nil || c.secretKey == "" {
		return ErrInvalidConfiguration
	}

	if !strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "//") ||
		strings.Contains(path, "://") {
		return ErrInvalidRequestPath
	}

	var requestBody io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode Paystack request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+path,
		requestBody,
	)
	if err != nil {
		return fmt.Errorf("create Paystack request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Paystack request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return fmt.Errorf("read Paystack response: %w", err)
	}
	if len(body) > maxResponseBodyBytes {
		return ErrResponseTooLarge
	}

	var envelope apiResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: decode response envelope", ErrInvalidResponse)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    envelope.Message,
		}
	}

	if !envelope.Status {
		return &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    envelope.Message,
		}
	}

	if result != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("%w: decode response data", ErrInvalidResponse)
		}
	}

	return nil
}
