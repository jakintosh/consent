package service

import (
	"errors"
	"net/http"
)

var (
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAccountNotFound        = errors.New("account not found")
	ErrServiceNotFound        = errors.New("service not found")
	ErrTokenInvalid           = errors.New("token invalid")
	ErrTokenNotFound          = errors.New("token not found")
	ErrInternal               = errors.New("internal error")
	ErrHandleExists           = errors.New("handle already exists")
	ErrInvalidHandle          = errors.New("invalid handle")
	ErrServiceExists          = errors.New("service already exists")
	ErrServiceProtected       = errors.New("service is protected")
	ErrInvalidService         = errors.New("invalid service")
	ErrInvalidUrl             = errors.New("invalid URL")
	ErrInvalidRedirect        = errors.New("invalid redirect URL")
	ErrInvalidScope           = errors.New("invalid scope")
	ErrMissingScope           = errors.New("missing scope")
	ErrIdentityScopeRequired  = errors.New("identity scope required")
	ErrInvalidScopeDependency = errors.New("invalid scope dependency")
	ErrInsufficientScope      = errors.New("insufficient scope")
	ErrAuthorizationDenied    = errors.New("authorization denied")
)

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrAccountNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, ErrServiceNotFound),
		errors.Is(err, ErrTokenInvalid),
		errors.Is(err, ErrTokenNotFound),
		errors.Is(err, ErrInvalidHandle),
		errors.Is(err, ErrInvalidScope),
		errors.Is(err, ErrMissingScope),
		errors.Is(err, ErrIdentityScopeRequired),
		errors.Is(err, ErrInvalidScopeDependency):
		return http.StatusBadRequest
	case errors.Is(err, ErrHandleExists),
		errors.Is(err, ErrServiceExists):
		return http.StatusConflict
	case errors.Is(err, ErrServiceProtected),
		errors.Is(err, ErrInsufficientScope):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidRedirect),
		errors.Is(err, ErrInvalidService):
		return http.StatusBadRequest
	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
