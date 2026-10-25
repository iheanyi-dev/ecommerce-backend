package use_cases

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// validateOrderProductLines rejects malformed product snapshots and lines
// that do not belong to one of the orders requested by the caller.
func validateOrderProductLines(
	lines []*entities.OrderProduct,
	allowedOrderIDs map[uuid.UUID]struct{},
) error {
	seenIDs := make(map[uuid.UUID]struct{}, len(lines))

	for _, line := range lines {
		if line == nil {
			return fmt.Errorf("repository returned a nil order product")
		}
		if line.ID == uuid.Nil || line.OrderID == uuid.Nil || line.ProductID == uuid.Nil {
			return fmt.Errorf("repository returned an order product with an invalid ID")
		}
		if _, ok := allowedOrderIDs[line.OrderID]; !ok {
			return fmt.Errorf("repository returned a product for an unexpected order")
		}
		if _, duplicate := seenIDs[line.ID]; duplicate {
			return fmt.Errorf("repository returned a duplicate order product ID")
		}
		seenIDs[line.ID] = struct{}{}

		if strings.TrimSpace(line.Name) == "" || line.Qty <= 0 || line.Price < 0 {
			return fmt.Errorf("repository returned an invalid order product")
		}
	}

	return nil
}
