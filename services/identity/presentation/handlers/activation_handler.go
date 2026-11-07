package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// ActivationHandler handles account activation HTTP requests.
type ActivationHandler struct {
	request *use_cases.RequestActivationUseCase
	resend  *use_cases.ResendActivationUseCase
	verify  *use_cases.VerifyEmailUseCase
}

// NewActivationHandler constructs the account activation handler.
func NewActivationHandler(
	request *use_cases.RequestActivationUseCase,
	resend *use_cases.ResendActivationUseCase,
	verify *use_cases.VerifyEmailUseCase,
) *ActivationHandler {
	return &ActivationHandler{
		request: request,
		resend:  resend,
		verify:  verify,
	}
}

// Request sends an account activation email.
func (h *ActivationHandler) Request(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body schemas.ActivationEmailRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.request.Execute(r.Context(), body.Email); err != nil {
		http.Error(w, "unable to request activation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// Resend sends another account activation email.
func (h *ActivationHandler) Resend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body schemas.ActivationEmailRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.resend.Execute(r.Context(), body.Email); err != nil {
		http.Error(w, "unable to resend activation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// Verify activates an account using its email verification token.
func (h *ActivationHandler) Verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body schemas.VerifyEmailRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.verify.Execute(r.Context(), body.Token); err != nil {
		if errors.Is(err, application_errors.ErrInvalidActivationToken) {
			http.Error(w, "invalid or expired activation token", http.StatusBadRequest)
			return
		}
		http.Error(w, "unable to verify email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
