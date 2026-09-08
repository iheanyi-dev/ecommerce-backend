package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// GetUserResponse represents the HTTP response returned when an
// authorized administrator retrieves a single user.
//
// Sensitive authentication information such as PasswordHash is
// deliberately excluded.
type GetUserResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewGetUserResponse maps the application-level GetUserResult into
// the HTTP response schema.
func NewGetUserResponse(
	result *dto.GetUserResult,
) GetUserResponse {
	return GetUserResponse{
		ID:        result.ID,
		FullName:  result.FullName,
		Email:     result.Email,
		Role:      result.Role,
		Status:    result.Status,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}
}
