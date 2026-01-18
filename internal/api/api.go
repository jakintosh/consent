// Package api provides RESTful HTTP handlers for authentication operations
// (login, logout, register, token refresh).
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type API struct {
	service *service.Service
}

func New(svc *service.Service) *API {
	return &API{
		service: svc,
	}
}

func decodeRequest[T any](req *T, w http.ResponseWriter, r *http.Request) bool {
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return false
	}
	return true
}

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrAccountNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, service.ErrServiceNotFound),
		errors.Is(err, service.ErrTokenInvalid),
		errors.Is(err, service.ErrTokenNotFound),
		errors.Is(err, service.ErrInvalidHandle):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrHandleExists):
		return http.StatusConflict
	case errors.Is(err, service.ErrInternal):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
