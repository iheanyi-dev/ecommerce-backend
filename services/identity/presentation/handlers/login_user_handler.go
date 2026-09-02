package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// LoginUserHandler handles HTTP requests for user authentication.
//
// The handler belongs to the presentation layer and therefore knows only
// about HTTP concerns and the application-layer authentication contract.
//
// It does not know how users are stored, how passwords are hashed, or how
// JWT tokens are generated. Those responsibilities remain behind the
// application ports.
type LoginUserHandler struct {
	authenticateUserService ports.AuthenticateUserService
}

// NewLoginUserHandler creates a new login HTTP handler.
//
// The application service is injected through the port so that the
// presentation layer remains independent of the concrete use case.
func NewLoginUserHandler(
	authenticateUserService ports.AuthenticateUserService,
) *LoginUserHandler {
	return &LoginUserHandler{
		authenticateUserService: authenticateUserService,
	}
}

// ServeHTTP implements http.Handler.
//
// Endpoint:
//
//	POST /api/v1/users/login
//
// The handler performs only presentation responsibilities:
//
//  1. Validate the HTTP method.
//  2. Decode the JSON request.
//  3. Convert the request into an application command.
//  4. Execute the authentication use case.
//  5. Convert the application result into an HTTP response.
func (h *LoginUserHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Authentication is performed using POST because credentials are
	// supplied in the request body.
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Decode the incoming JSON request.
	var request schemas.LoginUserRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	// Convert the presentation request into an application command.
	//
	// The application layer receives primitive values at its boundary and
	// is responsible for performing the actual authentication workflow.
	command := dto.LoginUserCommand{
		Email:    request.Email,
		Password: request.Password,
	}

	// Execute authentication through the application port.
	result, err := h.authenticateUserService.Authenticate(
		r.Context(),
		command,
	)

	if err != nil {
		// Authentication failures intentionally use generic responses.
		//
		// We must not reveal whether an email exists or whether the
		// password was incorrect.
		if errors.Is(err, use_cases.ErrInvalidCredentials) {
			http.Error(
				w,
				"invalid credentials",
				http.StatusUnauthorized,
			)
			return
		}

		// Inactive accounts are authenticated users whose account status
		// does not permit login.
		if errors.Is(err, use_cases.ErrAccountNotActive) {
			http.Error(
				w,
				"account is not active",
				http.StatusUnauthorized,
			)
			return
		}

		// Token generation failure is an internal authentication failure.
		// Do not expose JWT/infrastructure details to the client.
		if errors.Is(err, use_cases.ErrTokenGeneration) {
			http.Error(
				w,
				"failed to authenticate user",
				http.StatusInternalServerError,
			)
			return
		}

		// Any unexpected application/infrastructure failure is also
		// represented as an internal server error.
		http.Error(
			w,
			"failed to authenticate user",
			http.StatusInternalServerError,
		)
		return
	}

	// Convert the application result into the presentation response.
	response := schemas.LoginUserResponse{
		ID:          result.ID,
		Email:       result.Email,
		Role:        result.Role,
		Status:      result.Status,
		AccessToken: result.AccessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// The response headers have already been written at this point,
		// so there is no useful HTTP status change we can make.
		return
	}
}