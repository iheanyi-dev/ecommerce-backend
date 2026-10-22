package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/errors"
)

func customerID(r *http.Request) (uuid.UUID, error) {
	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok || identity.UserID == uuid.Nil {
		return uuid.Nil, presentationerrors.ErrAuthenticationRequired
	}
	return identity.UserID, nil
}

func parsePathUUID(r *http.Request, key string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(key))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, presentationerrors.ErrInvalidRequestBody
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return presentationerrors.ErrInvalidRequestBody
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return presentationerrors.ErrInvalidRequestBody
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	presentationerrors.WriteError(w, err)
}
