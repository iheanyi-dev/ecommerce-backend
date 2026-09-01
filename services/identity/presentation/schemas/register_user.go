package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// RegisterUserRequest represents the JSON payload accepted by the user
// registration endpoint.
//
// This schema belongs to the presentation layer because it represents the
// external HTTP API contract.
type RegisterUserRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterUserResponse represents the public response returned after a
// successful user registration.
//
// Sensitive information such as the user's password and password hash is
// deliberately excluded from this response.
type RegisterUserResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewRegisterUserResponse converts the application result into the HTTP
// response schema.
func NewRegisterUserResponse(
	result dto.RegisterUserResult,
) RegisterUserResponse {
	return RegisterUserResponse{
		ID:        result.ID,
		FullName:  result.FullName,
		Email:     result.Email,
		Role:      result.Role,
		Status:    result.Status,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}
}
