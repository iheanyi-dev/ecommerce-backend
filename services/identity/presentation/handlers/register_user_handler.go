package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
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
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	var request schemas.RegisterUserRequest

	// Decode the incoming JSON body into the presentation schema.
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid request body",
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
		h.handleError(w, err)
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

// handleError maps known application and domain errors into appropriate
// HTTP responses.
//
// The presentation layer decides how application and domain errors are
// represented over HTTP. The application/domain layers remain unaware
// of HTTP status codes.
func (h *RegisterUserHandler) handleError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, use_cases.ErrEmailAlreadyExists):
		writeJSONError(
			w,
			http.StatusConflict,
			"email already exists",
		)

	case errors.Is(err, user.ErrInvalidEmail):
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid email address",
		)

	case errors.Is(err, user.ErrInvalidFullName):
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid full name",
		)

	default:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"failed to register user",
		)
	}
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