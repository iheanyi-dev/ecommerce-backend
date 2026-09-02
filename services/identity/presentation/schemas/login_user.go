package schemas

// LoginUserRequest represents the JSON payload submitted when
// a user attempts to authenticate.
//
// These fields intentionally remain primitive values because
// presentation is responsible only for transporting HTTP data.
// Authentication and validation are handled by the application layer.
type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginUserResponse represents the successful authentication response.
//
// Password information is never returned to the client.
type LoginUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
}