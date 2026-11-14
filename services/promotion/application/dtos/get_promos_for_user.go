package dtos

// GetPromosForUserInput contains the customer and pagination criteria.
type GetPromosForUserInput struct {
	CustomerID string
	Limit      int
	Offset     int
}
