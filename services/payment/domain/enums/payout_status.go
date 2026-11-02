package enums

type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusSucceeded  PayoutStatus = "succeeded"
	PayoutStatusFailed     PayoutStatus = "failed"
)

func (s PayoutStatus) IsValid() bool {
	switch s {
	case PayoutStatusPending,
		PayoutStatusProcessing,
		PayoutStatusSucceeded,
		PayoutStatusFailed:
		return true
	default:
		return false
	}
}
