package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// ListUsersResponse represents the HTTP response returned by the
// administrative user-listing endpoint.
//
// Only non-sensitive user information is exposed.
type ListUsersResponse struct {
	Users []UserSummaryResponse `json:"users"`
}

// UserSummaryResponse represents a safe user representation for
// administrative user management.
type UserSummaryResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewListUsersResponse maps the application result into the HTTP response.
func NewListUsersResponse(
	result *dto.ListUsersResult,
) ListUsersResponse {
	users := make([]UserSummaryResponse, 0, len(result.Users))

	for _, existingUser := range result.Users {
		users = append(users, UserSummaryResponse{
			ID:        existingUser.ID,
			FullName:  existingUser.FullName,
			Email:     existingUser.Email,
			Role:      existingUser.Role,
			Status:    existingUser.Status,
			CreatedAt: existingUser.CreatedAt,
			UpdatedAt: existingUser.UpdatedAt,
		})
	}

	return ListUsersResponse{
		Users: users,
	}
}
