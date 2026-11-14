package events

// Promotion event names are owned by the Promotion service. Product mirrors
// the transport contract here to avoid importing another service's internals.
const (
	PromotionCreated   = "promotion.created"
	PromotionActivated = "promotion.activated"
	PromotionRenewed   = "promotion.renewed"
	PromotionCancelled = "promotion.cancelled"
	PromotionExpired   = "promotion.expired"
)

// PromotionEvent is the Product-facing promotion snapshot published by
// Promotion whenever a promotion is created or changes lifecycle state.
type PromotionEvent struct {
	PromoID   string  `json:"promo_id"`
	StoreID   string  `json:"store_id"`
	ProductID *string `json:"product_id"`
	Status    string  `json:"status"`
}
