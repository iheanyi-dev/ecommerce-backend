package middleware

import (
	"crypto/subtle"
	"net/http"

	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/errors"
)

const (
	ServiceNameHeader   = "X-Service-Name"
	ServiceSecretHeader = "X-Service-Secret"
)

// ServiceAuthentication protects the Storage HTTP boundary.
//
// Storage is an internal-only service. Even though GetPublicObject exposes
// business-level public objects, Storage itself must never be directly public.
// The Gateway authenticates to Storage using these trusted service credentials.
func ServiceAuthentication(serviceName, serviceSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedName := r.Header.Get(ServiceNameHeader)
			providedSecret := r.Header.Get(ServiceSecretHeader)

			if !secureEqual(providedName, serviceName) ||
				!secureEqual(providedSecret, serviceSecret) {
				presentation_errors.WriteError(
					w,
					presentation_errors.ErrInvalidServiceCredentials,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func secureEqual(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(provided),
		[]byte(expected),
	) == 1
}
