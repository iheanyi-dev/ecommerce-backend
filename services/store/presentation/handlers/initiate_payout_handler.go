package handlers

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
)

// InitiatePayoutHandler handles payout initiation requests from authenticated
// Store owners. Store resolves ownership; the caller cannot supply a Store ID.
type InitiatePayoutHandler struct {
	service ports.InitiatePayoutService
}

// NewInitiatePayoutHandler constructs the payout initiation HTTP handler.
func NewInitiatePayoutHandler(
	service ports.InitiatePayoutService,
) *InitiatePayoutHandler {
	return &InitiatePayoutHandler{service: service}
}

func (h *InitiatePayoutHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}

	var request struct {
		OrderID       string `json:"order_id"`
		AccountName   string `json:"account_name"`
		AccountNumber string `json:"account_number"`
		BankCode      string `json:"bank_code"`
		BankName      string `json:"bank_name"`
		CountryCode   string `json:"country_code"`
	}

	if err := decodeJSON(r, &request); err != nil {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequestBody,
		)
		return
	}

	orderID, err := uuid.Parse(strings.TrimSpace(request.OrderID))
	if err != nil || orderID == uuid.Nil {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequestBody,
		)
		return
	}

	beneficiary := ports.PayoutBeneficiary{
		AccountName:   strings.TrimSpace(request.AccountName),
		AccountNumber: strings.TrimSpace(request.AccountNumber),
		BankCode:      strings.TrimSpace(request.BankCode),
		BankName:      strings.TrimSpace(request.BankName),
		CountryCode:   strings.TrimSpace(request.CountryCode),
	}

	if beneficiary.AccountName == "" ||
		beneficiary.AccountNumber == "" ||
		beneficiary.BankCode == "" ||
		beneficiary.BankName == "" ||
		beneficiary.CountryCode == "" {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequestBody,
		)
		return
	}

	result, err := h.service.Execute(r.Context(), orderID, beneficiary)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	response := struct {
		PayoutID      uuid.UUID `json:"payout_id"`
		AttemptID     uuid.UUID `json:"attempt_id"`
		Reference     string    `json:"reference"`
		Status        string    `json:"status"`
		AttemptStatus string    `json:"attempt_status"`
		Provider      string    `json:"provider"`
		AmountMinor   int64     `json:"amount_minor"`
		Currency      string    `json:"currency"`
	}{
		PayoutID:      result.PayoutID,
		AttemptID:     result.AttemptID,
		Reference:     result.Reference,
		Status:        result.Status,
		AttemptStatus: result.AttemptStatus,
		Provider:      result.Provider,
		AmountMinor:   result.AmountMinor,
		Currency:      result.Currency,
	}

	writeJSON(w, http.StatusOK, response)
}
