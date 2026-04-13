// Package middleware provides HTTP middleware for request validation and processing.
package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/validation"
)

// ValidateUUIDMiddleware validates that the uuid URL parameter is a well-formed UUID.
// It reads the "uuid" Chi URL parameter and returns 400 Bad Request if it is absent or malformed.
//
// Usage:
//
//	r.Route("/{uuid}", func(r chi.Router) {
//	    r.Use(middleware.ValidateUUIDMiddleware)
//	    r.Get("/", handler.Get)
//	})
func ValidateUUIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		UUID := chi.URLParam(r, "uuid")

		if UUID == "" {
			response.RespondError(w, http.StatusBadRequest, "valid UUID is required", "")
			return
		}

		if err := validation.ValidateUUID(UUID); err != nil {
			response.RespondError(w, http.StatusBadRequest, "invalid UUID format", err.Error())
			return
		}

		next.ServeHTTP(w, r)
	})
}
