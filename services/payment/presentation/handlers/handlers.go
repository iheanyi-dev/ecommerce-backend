package handlers

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/schemas"
)

type Handlers struct {
	InitiatePayment http.Handler
	VerifyPayment   http.Handler
	GetPayment      http.Handler
	ListPayments    http.Handler

	InitiatePayout http.Handler
	VerifyPayout   http.Handler
	GetPayout      http.Handler
	ListPayouts    http.Handler
	RetryPayout    http.Handler
}

// New constructs the HTTP handlers around the already-wired application use cases.
func New(useCases *infrastructure.UseCases) *Handlers {
	return &Handlers{
		InitiatePayment: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			initiatePayment(w, r, useCases.InitiatePayment)
		}),
		VerifyPayment: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifyPayment(w, r, useCases.VerifyPayment)
		}),
		GetPayment: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getPayment(w, r, useCases.GetPayment)
		}),
		ListPayments: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			listPayments(w, r, useCases.ListPayments)
		}),
		InitiatePayout: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			initiatePayout(w, r, useCases.InitiatePayout)
		}),
		VerifyPayout: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifyPayout(w, r, useCases.VerifyPayout)
		}),
		GetPayout: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getPayout(w, r, useCases.GetPayout)
		}),
		ListPayouts: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			listPayouts(w, r, useCases.ListPayouts)
		}),
		RetryPayout: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retryPayout(w, r, useCases.RetryPayout)
		}),
	}
}

func initiatePayment(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.InitiatePaymentUseCase,
) {
	var request schemas.InitiatePaymentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	purpose := enums.PaymentPurpose(strings.TrimSpace(request.Purpose))
	if !purpose.IsValid() {
		writeError(w, application.ErrInvalidProviderResponse)
		return
	}

	amount, err := value_objects.NewMoney(request.Amount, request.Currency)
	if err != nil {
		writeError(w, err)
		return
	}

	output, err := uc.Execute(r.Context(), use_cases.InitiatePaymentInput{
		PayerID:         request.PayerID,
		Purpose:         purpose,
		BusinessRefType: request.BusinessRefType,
		BusinessRefID:   request.BusinessRefID,
		Amount:          amount,
		CustomerEmail:   request.CustomerEmail,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, schemas.InitiatePaymentResponse{
		PaymentID:     output.PaymentID,
		AttemptID:     output.AttemptID,
		PaymentStatus: string(output.PaymentStatus),
		AttemptStatus: string(output.AttemptStatus),
		Provider:      output.Provider,
		CheckoutURL:   output.CheckoutURL,
		Amount:        output.Amount,
		Currency:      output.Currency,
	})
}

func verifyPayment(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.VerifyPaymentUseCase,
) {
	details, err := uc.Execute(r.Context(), r.PathValue("paymentID"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.MapPaymentDetails(details))
}

func getPayment(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.GetPaymentByIDUseCase,
) {
	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok {
		writeError(w, application.ErrPaymentNotFound)
		return
	}

	details, err := uc.Execute(r.Context(), r.PathValue("paymentID"))
	if err != nil {
		writeError(w, err)
		return
	}

	// Do not reveal whether a payment exists outside the authenticated payer's scope.
	if details.PayerID != identity.UserID.String() {
		writeError(w, application.ErrPaymentNotFound)
		return
	}

	writeJSON(w, http.StatusOK, schemas.MapPaymentDetails(details))
}

func listPayments(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.ListPaymentsByPayerIDUseCase,
) {
	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok {
		writeError(w, application.ErrPaymentNotFound)
		return
	}

	page, err := parsePositiveQuery(r, "page", 1)
	if err != nil {
		writeError(w, application.ErrInvalidPagination)
		return
	}
	pageSize, err := parsePositiveQuery(r, "page_size", use_cases.DefaultPaymentPageSize)
	if err != nil || pageSize > use_cases.MaxPaymentPageSize {
		writeError(w, application.ErrInvalidPagination)
		return
	}

	output, err := uc.Execute(r.Context(), use_cases.ListPaymentsByPayerIDInput{
		PayerID:  identity.UserID.String(),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	response := schemas.PaymentListResponse{
		Items:      make([]schemas.PaymentResponse, 0, len(output.Items)),
		Page:       output.Page,
		PageSize:   output.PageSize,
		Total:      output.Total,
		TotalPages: output.TotalPages,
	}
	for _, item := range output.Items {
		response.Items = append(response.Items, schemas.MapPaymentDetails(item))
	}

	writeJSON(w, http.StatusOK, response)
}

func initiatePayout(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.InitiatePayoutUseCase,
) {
	var request schemas.InitiatePayoutRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	reference, err := value_objects.NewBusinessReference(
		request.BusinessReferenceType,
		request.BusinessReferenceID,
	)
	if err != nil {
		writeError(w, err)
		return
	}

	amount, err := value_objects.NewMoney(request.Amount, request.Currency)
	if err != nil {
		writeError(w, err)
		return
	}

	output, err := uc.Execute(r.Context(), ports.PayoutRequest{
		BusinessReference: reference,
		StoreID:           request.StoreID,
		OrderID:           request.OrderID,
		Reason:            request.Reason,
		Amount:            amount,
		Beneficiary:       request.Beneficiary,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, schemas.MapPayoutOperation(output))
}

func verifyPayout(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.VerifyPayoutUseCase,
) {
	output, err := uc.Execute(r.Context(), r.PathValue("payoutID"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.MapPayoutOperation(output))
}

func getPayout(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.GetPayoutUseCase,
) {
	storeID := strings.TrimSpace(r.URL.Query().Get("store_id"))
	if storeID == "" {
		writeError(w, application.ErrPayoutNotFound)
		return
	}

	payout, err := uc.Execute(r.Context(), r.PathValue("payoutID"), storeID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.MapPayout(payout))
}

func listPayouts(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.ListPayoutsByStoreIDUseCase,
) {
	storeID := strings.TrimSpace(r.URL.Query().Get("store_id"))
	if storeID == "" {
		writeError(w, application.ErrInvalidPagination)
		return
	}

	limit, err := parsePositiveQuery(r, "limit", 20)
	if err != nil || limit > 100 {
		writeError(w, application.ErrInvalidPagination)
		return
	}
	offset, err := parseNonNegativeQuery(r, "offset", 0)
	if err != nil {
		writeError(w, application.ErrInvalidPagination)
		return
	}

	output, err := uc.Execute(r.Context(), storeID, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}

	response := schemas.PayoutListResponse{
		Items:  make([]schemas.PayoutResponse, 0, len(output.Payouts)),
		Total:  output.Total,
		Limit:  output.Limit,
		Offset: output.Offset,
	}
	for _, payout := range output.Payouts {
		response.Items = append(response.Items, schemas.MapPayout(payout))
	}

	writeJSON(w, http.StatusOK, response)
}

func retryPayout(
	w http.ResponseWriter,
	r *http.Request,
	uc *use_cases.RetryPayoutUseCase,
) {
	var request struct {
		StoreID string `json:"store_id"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	output, err := uc.Execute(
		r.Context(),
		r.PathValue("payoutID"),
		strings.TrimSpace(request.StoreID),
	)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.MapPayoutOperation(output))
}
