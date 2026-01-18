package service

import (
	"errors"
	"net/http"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountNotFound    = errors.New("account not found")
	ErrServiceNotFound    = errors.New("service not found")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenNotFound      = errors.New("token not found")
	ErrInternal           = errors.New("internal error")
	ErrHandleExists       = errors.New("handle already exists")
	ErrInvalidHandle      = errors.New("invalid handle")
)

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrAccountNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, ErrServiceNotFound),
		errors.Is(err, ErrTokenInvalid),
		errors.Is(err, ErrTokenNotFound),
		errors.Is(err, ErrInvalidHandle):
		return http.StatusBadRequest
	case errors.Is(err, ErrHandleExists):
		return http.StatusConflict
	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
