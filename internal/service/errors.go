package service

import (
	"errors"
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
