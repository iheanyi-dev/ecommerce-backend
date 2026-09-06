package schemas

import "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"

// UpdateUserProfileRequest represents the HTTP request body for updating
// the authenticated user's profile.
type UpdateUserProfileRequest struct {
	FullName string `json:"full_name"`
}

// UpdateUserProfileResponse represents the updated authenticated user's
// profile returned by the API.
type UpdateUserProfileResponse struct {
	UserID    string `json:"user_id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// NewUpdateUserProfileResponse maps the application result to the HTTP
// response schema.
func NewUpdateUserProfileResponse(
	result dto.UpdateUserProfileResult,
) UpdateUserProfileResponse {
	return UpdateUserProfileResponse{
		UserID:    result.UserID,
		FullName:  result.FullName,
		Email:     result.Email,
		Role:      result.Role,
		Status:    result.Status,
		UpdatedAt: result.UpdatedAt,
	}
}
