package ports

import "context"

type PromotionEventPublisher interface {
	Publish(ctx context.Context, event PendingPromotionEvent) error
}
