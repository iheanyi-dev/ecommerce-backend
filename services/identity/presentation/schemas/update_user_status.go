package schemas

import "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"

// UpdateUserStatusRequest represents the HTTP request body for changing a
// user's account status.
type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// UpdateUserStatusResponse represents the result of an administrative
// account-status update.
type UpdateUserStatusResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

// NewUpdateUserStatusResponse maps the application result to the HTTP
// response schema.
func NewUpdateUserStatusResponse(
	result dto.UpdateUserStatusResult,
) UpdateUserStatusResponse {
	return UpdateUserStatusResponse{
		UserID: result.UserID,
		Status: result.Status,
	}
}
