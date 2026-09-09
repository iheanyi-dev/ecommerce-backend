package handlers

import (
	"net/http"
	"strconv"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

const (
	defaultListUsersPage  = 1
	defaultListUsersLimit = 20
)

// ListUsersHandler handles administrative requests to retrieve users.
//
// Authentication and authorization are handled by middleware before this
// handler is reached.
//
// The handler is responsible for HTTP concerns only:
//   - validating the HTTP method
//   - validating pagination parameters
//   - calling the application service
//   - returning the application result as JSON
type ListUsersHandler struct {
	listUsersService ports.ListUsersService
}

// NewListUsersHandler creates a new ListUsersHandler.
func NewListUsersHandler(
	listUsersService ports.ListUsersService,
) *ListUsersHandler {
	return &ListUsersHandler{
		listUsersService: listUsersService,
	}
}

// ServeHTTP handles:
//
//	GET /api/v1/admin/users?page=1&limit=20
func (h *ListUsersHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}

	// AuthenticationMiddleware should normally guarantee that an
	// authenticated identity exists here.
	//
	// This defensive check prevents the handler from accidentally
	// processing an unauthenticated request if the route is misconfigured.
	if _, ok := middleware.AuthenticatedIdentity(r.Context()); !ok {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrAuthenticationRequired,
		)
		return
	}

	page, limit, paginationErr := parseListUsersPagination(r)
	if paginationErr != "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			paginationErr,
		)
		return
	}

	// Convert the one-based page number used by HTTP clients into the
	// zero-based offset expected by the application/repository layer.
	offset := (page - 1) * limit

	result, err := h.listUsersService.Execute(
		r.Context(),
		limit,
		offset,
	)
	if err != nil {
		// Delegate application and domain error translation to the common
		// presentation error translator so every endpoint uses the same
		// HTTP status codes and JSON error response format.
		presentation_errors.WriteError(w, err)
		return
	}

	response := schemas.NewListUsersResponse(result)

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}

// parseListUsersPagination parses and validates the page and limit
// query parameters.
//
// Defaults:
//
//	page  = 1
//	limit = 20
//
// Page numbers are one-based.
func parseListUsersPagination(
	r *http.Request,
) (int, int, string) {
	page := defaultListUsersPage
	limit := defaultListUsersLimit

	query := r.URL.Query()

	if value := query.Get("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)

		if err != nil || parsedPage <= 0 {
			return 0, 0, "invalid page"
		}

		page = parsedPage
	}

	if value := query.Get("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)

		if err != nil || parsedLimit <= 0 {
			return 0, 0, "invalid limit"
		}

		limit = parsedLimit
	}

	return page, limit, ""
}
