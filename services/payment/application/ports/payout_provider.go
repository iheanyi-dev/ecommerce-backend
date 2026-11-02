package ports

import (
	"context"
	"encoding/json"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// PayoutProvider abstracts provider-specific transfers from application logic.
type PayoutProvider interface {
	InitiatePayout(ctx context.Context, request PayoutInitiationRequest) (PayoutInitiationResult, error)
	ReconcilePayout(ctx context.Context, request PayoutReconciliationRequest) (PayoutReconciliationResult, error)
}

type PayoutInitiationRequest struct {
	Payout  *entities.Payout
	Attempt *entities.PayoutAttempt
}

type PayoutInitiationResult struct {
	ProviderReference string
	Status            enums.PayoutAttemptStatus
	Metadata          json.RawMessage
}

type PayoutReconciliationRequest struct {
	Payout  *entities.Payout
	Attempt *entities.PayoutAttempt
}

type PayoutReconciliationResult struct {
	Amount            int64
	Currency          string
	ProviderReference string
	Status            enums.PayoutAttemptStatus
	FailureReason     string
	Metadata          json.RawMessage
}

// PayoutRequest is the trusted application input received from Order.
type PayoutRequest struct {
	BusinessReference value_objects.BusinessReference
	StoreID           string
	OrderID           string
	Reason            string
	Amount            value_objects.Money
	Beneficiary       entities.PayoutBeneficiary
}
