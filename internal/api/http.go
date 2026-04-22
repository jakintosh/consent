package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func decodeRequest[T any](r *http.Request) (T, error) {
	var req T
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrAccountNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, service.ErrServiceNotFound),
		errors.Is(err, service.ErrTokenInvalid),
		errors.Is(err, service.ErrTokenNotFound),
		errors.Is(err, service.ErrInvalidHandle),
		errors.Is(err, service.ErrInvalidScope),
		errors.Is(err, service.ErrMissingScope),
		errors.Is(err, service.ErrIdentityScopeRequired),
		errors.Is(err, service.ErrInvalidScopeDependency):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrHandleExists),
		errors.Is(err, service.ErrServiceExists):
		return http.StatusConflict
	case errors.Is(err, service.ErrServiceProtected),
		errors.Is(err, service.ErrInsufficientScope):
		return http.StatusForbidden
	case errors.Is(err, service.ErrInvalidRedirect),
		errors.Is(err, service.ErrInvalidService):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrInternal):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
