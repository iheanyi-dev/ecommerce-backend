package http

import (
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
)

const maxProviderWebhookBody = 1 << 20 // 1 MiB

type ProviderWebhookHandler struct {
	useCase *use_cases.HandleProviderWebhookUseCase
}

func NewProviderWebhookHandler(
	useCase *use_cases.HandleProviderWebhookUseCase,
) *ProviderWebhookHandler {
	return &ProviderWebhookHandler{useCase: useCase}
}

func (h *ProviderWebhookHandler) ServeHTTP(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		nethttp.Error(w, "method not allowed", nethttp.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.useCase == nil {
		nethttp.Error(w, "webhook handler unavailable", nethttp.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(nethttp.MaxBytesReader(w, r.Body, maxProviderWebhookBody))
	if err != nil {
		nethttp.Error(w, "invalid webhook body", nethttp.StatusBadRequest)
		return
	}

	headers := make(map[string]string, len(r.Header))
	for name, values := range r.Header {
		headers[name] = strings.Join(values, ",")
	}

	result, err := h.useCase.Execute(r.Context(), body, headers)
	if err != nil {
		if errors.Is(err, ports.ErrInvalidWebhookSignature) {
			nethttp.Error(w, "invalid webhook signature", nethttp.StatusUnauthorized)
			return
		}

		// Invalid provider payloads are permanent client errors. Other failures
		// may be transient and must remain retryable by the provider.
		if errors.Is(err, application.ErrInvalidProviderResponse) {
			nethttp.Error(w, "invalid webhook event", nethttp.StatusBadRequest)
			return
		}
		nethttp.Error(w, "webhook processing failed", nethttp.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
