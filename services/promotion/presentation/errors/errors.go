package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type errorResponse struct {
	Error string `json:"error"`
}

var (
	ErrAuthenticationRequired    = stderrors.New("authentication required")
	ErrForbidden                 = stderrors.New("forbidden")
	ErrInvalidRequestBody        = stderrors.New("invalid request body")
	ErrMethodNotAllowed          = stderrors.New("method not allowed")
	ErrInvalidServiceCredentials = stderrors.New("invalid service credentials")
)

func WriteError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case stderrors.Is(err, apperrors.ErrPromotionNotFound):
		statusCode, message = http.StatusNotFound, "promotion not found"
	case stderrors.Is(err, apperrors.ErrPromotionConflict):
		statusCode, message = http.StatusConflict, "promotion conflict"
	case stderrors.Is(err, apperrors.ErrInvalidPromotionQuery):
		statusCode, message = http.StatusBadRequest, "invalid promotion query"
	case stderrors.Is(err, apperrors.ErrPromotionNotDeletable):
		statusCode, message = http.StatusConflict, "promotion cannot be deleted"
	case stderrors.Is(err, domainerrors.ErrInvalidPromotion):
		statusCode, message = http.StatusBadRequest, "invalid promotion"
	case stderrors.Is(err, domainerrors.ErrPromotionAlreadyPaused),
		stderrors.Is(err, domainerrors.ErrPromotionNotPaused),
		stderrors.Is(err, domainerrors.ErrPromotionExpired),
		stderrors.Is(err, domainerrors.ErrPromotionCancelled),
		stderrors.Is(err, domainerrors.ErrPromotionAlreadyActive),
		stderrors.Is(err, domainerrors.ErrPromotionNotActive),
		stderrors.Is(err, domainerrors.ErrPromotionAlreadyExpired),
		stderrors.Is(err, domainerrors.ErrPromotionNotExpired),
		stderrors.Is(err, domainerrors.ErrPromotionPaused):
		statusCode, message = http.StatusConflict, "invalid promotion state"
	case stderrors.Is(err, ErrAuthenticationRequired):
		statusCode, message = http.StatusUnauthorized, "authentication required"
	case stderrors.Is(err, ErrForbidden):
		statusCode, message = http.StatusForbidden, "forbidden"
	case stderrors.Is(err, ErrInvalidRequestBody):
		statusCode, message = http.StatusBadRequest, "invalid request body"
	case stderrors.Is(err, ErrMethodNotAllowed):
		statusCode, message = http.StatusMethodNotAllowed, "method not allowed"
	case stderrors.Is(err, ErrInvalidServiceCredentials):
		statusCode, message = http.StatusUnauthorized, "invalid service credentials"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func ToGRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, apperrors.ErrPromotionNotFound):
		return status.Error(codes.NotFound, "promotion not found")
	case stderrors.Is(err, apperrors.ErrPromotionConflict):
		return status.Error(codes.AlreadyExists, "promotion conflict")
	case stderrors.Is(err, apperrors.ErrInvalidPromotionQuery),
		stderrors.Is(err, domainerrors.ErrInvalidPromotion):
		return status.Error(codes.InvalidArgument, "invalid promotion request")
	case stderrors.Is(err, apperrors.ErrPromotionNotDeletable),
		stderrors.Is(err, domainerrors.ErrPromotionAlreadyPaused),
		stderrors.Is(err, domainerrors.ErrPromotionNotPaused),
		stderrors.Is(err, domainerrors.ErrPromotionExpired),
		stderrors.Is(err, domainerrors.ErrPromotionCancelled),
		stderrors.Is(err, domainerrors.ErrPromotionAlreadyActive),
		stderrors.Is(err, domainerrors.ErrPromotionNotActive),
		stderrors.Is(err, domainerrors.ErrPromotionAlreadyExpired),
		stderrors.Is(err, domainerrors.ErrPromotionNotExpired),
		stderrors.Is(err, domainerrors.ErrPromotionPaused):
		return status.Error(codes.FailedPrecondition, "invalid promotion state")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
