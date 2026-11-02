package enums

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending,
		PaymentStatusProcessing,
		PaymentStatusSucceeded,
		PaymentStatusFailed:
		return true
	default:
		return false
	}
}
