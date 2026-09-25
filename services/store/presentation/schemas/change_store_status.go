package schemas

// ChangeStoreStatusRequest contains the status requested by a trusted
// internal service such as Billing/Subscription.
type ChangeStoreStatusRequest struct {
	Status string `json:"status"`
}
