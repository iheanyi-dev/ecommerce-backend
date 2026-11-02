package enums

type PaymentPurpose string

const (
	PaymentPurposeOrder        PaymentPurpose = "order"
	PaymentPurposeSubscription PaymentPurpose = "subscription"
)

func (p PaymentPurpose) IsValid() bool {
	switch p {
	case PaymentPurposeOrder, PaymentPurposeSubscription:
		return true
	default:
		return false
	}
}
