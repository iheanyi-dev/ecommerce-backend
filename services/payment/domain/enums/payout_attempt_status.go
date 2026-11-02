package enums

type PayoutAttemptStatus string

const (
	PayoutAttemptStatusPending                PayoutAttemptStatus = "pending"
	PayoutAttemptStatusProcessing             PayoutAttemptStatus = "processing"
	PayoutAttemptStatusSucceeded              PayoutAttemptStatus = "succeeded"
	PayoutAttemptStatusFailed                 PayoutAttemptStatus = "failed"
	PayoutAttemptStatusRequiresReconciliation PayoutAttemptStatus = "requires_reconciliation"
)

func (s PayoutAttemptStatus) IsValid() bool {
	switch s {
	case PayoutAttemptStatusPending,
		PayoutAttemptStatusProcessing,
		PayoutAttemptStatusSucceeded,
		PayoutAttemptStatusFailed,
		PayoutAttemptStatusRequiresReconciliation:
		return true
	default:
		return false
	}
}
