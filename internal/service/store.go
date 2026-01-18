package service

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// Store handles persistence of identity data, refresh tokens, and services.
type Store interface {
	InsertIdentity(handle string, secret []byte) error
	GetSecret(handle string) ([]byte, error)

	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (handle string, err error)

	InsertService(name, display, audience, redirect string) error
	GetService(name string) (ServiceDefinition, error)
	UpdateService(name, display, audience, redirect string) error
	DeleteService(name string) (deleted bool, err error)
	ListServices() ([]ServiceDefinition, error)
}
