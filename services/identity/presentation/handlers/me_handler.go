package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// MeHandler handles requests to the authenticated user's endpoint.
//
// Authentication itself is performed by AuthenticationMiddleware before
// this handler is reached.
//
// Therefore, this handler does not inspect JWTs, authorization headers,
// signatures, or token expiration. Its only responsibility is to retrieve
// the authenticated identity that the middleware placed into the request
// context.
type MeHandler struct{}

// NewMeHandler creates a new authenticated-user handler.
func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

// ServeHTTP returns the identity of the currently authenticated user.
//
// The endpoint is protected by AuthenticationMiddleware.
//
// Expected endpoint:
//
//	GET /api/v1/users/me
func (h *MeHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	// This endpoint is intentionally read-only.
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// AuthenticationMiddleware should always place an authenticated
	// identity into the request context before this handler executes.
	//
	// We still check the value defensively rather than assuming it exists.
	identity, ok := middleware.AuthenticatedIdentity(
		r.Context(),
	)

	if !ok {
		// This normally indicates that the route was incorrectly configured
		// without AuthenticationMiddleware.
		http.Error(
			w,
			"authentication required",
			http.StatusUnauthorized,
		)
		return
	}

	response := struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}{
		UserID: identity.UserID,
		Role:   identity.Role,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	// The response only contains authenticated identity information.
	//
	// Passwords, password hashes, JWT secrets, or other security-sensitive
	// information are never exposed.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}