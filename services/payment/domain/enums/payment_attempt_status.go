package enums

type PaymentAttemptStatus string

const (
	PaymentAttemptStatusPending                PaymentAttemptStatus = "pending"
	PaymentAttemptStatusProcessing             PaymentAttemptStatus = "processing"
	PaymentAttemptStatusSucceeded              PaymentAttemptStatus = "succeeded"
	PaymentAttemptStatusFailed                 PaymentAttemptStatus = "failed"
	PaymentAttemptStatusRequiresReconciliation PaymentAttemptStatus = "requires_reconciliation"
)

func (s PaymentAttemptStatus) IsValid() bool {
	switch s {
	case PaymentAttemptStatusPending,
		PaymentAttemptStatusProcessing,
		PaymentAttemptStatusSucceeded,
		PaymentAttemptStatusFailed,
		PaymentAttemptStatusRequiresReconciliation:
		return true
	default:
		return false
	}
}
