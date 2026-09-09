package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// RegisterUserHandler handles HTTP requests for user registration.
//
// The handler belongs to the presentation layer and is responsible only for
// HTTP concerns:
//   - validating the HTTP method
//   - decoding JSON
//   - converting HTTP input into an application command
//   - invoking the application service
//   - converting the application result into an HTTP response
//
// It does not perform domain validation, password hashing, or database
// operations.
type RegisterUserHandler struct {
	registerUserService ports.RegisterUserService
}

// NewRegisterUserHandler creates a new registration HTTP handler.
func NewRegisterUserHandler(
	registerUserService ports.RegisterUserService,
) *RegisterUserHandler {
	return &RegisterUserHandler{
		registerUserService: registerUserService,
	}
}

// ServeHTTP implements http.Handler.
func (h *RegisterUserHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Registration is only available through POST.
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}
	var request schemas.RegisterUserRequest

	// Decode the incoming JSON body into the presentation schema.
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequestBody,
		)
		return
	}
	// Convert the presentation schema into the application command.
	command := dto.RegisterUserCommand{
		FullName: request.FullName,
		Email:    request.Email,
		Password: request.Password,
	}

	// Execute the registration application workflow.
	result, err := h.registerUserService.Execute(
		r.Context(),
		command,
	)
	if err != nil {
		// Delegate application and domain error translation to the common
		// presentation error translator so every endpoint uses the same
		// HTTP status codes and JSON error response format.
		presentation_errors.WriteError(w, err)
		return
	}
	// Convert the application result into the HTTP response schema.
	response := schemas.NewRegisterUserResponse(result)

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

// writeJSON writes a successful JSON response.
func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// At this point the response is already being written. There is no useful
	// HTTP-level action available if JSON encoding fails, so the error is
	// intentionally ignored.
	_ = json.NewEncoder(w).Encode(data)
}

// writeJSONError writes a consistent JSON error response.
func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		w,
		status,
		map[string]string{
			"error": message,
		},
	)
}
