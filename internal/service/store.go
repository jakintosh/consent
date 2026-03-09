package service

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type IdentityRecord struct {
	Subject string
	Handle  string
	Secret  []byte
}

// Store handles persistence of identity data, refresh tokens, and services.
type Store interface {
	InsertIdentity(subject, handle string, secret []byte) error
	GetIdentityByHandle(handle string) (IdentityRecord, error)
	GetIdentityBySubject(subject string) (IdentityRecord, error)

	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (subject string, err error)

	ListGrantedScopeNames(subject, service string) ([]string, error)
	InsertGrants(subject, service string, scopes []string) error

	InsertService(name, display, audience, redirect string) error
	UpsertSystemServices(services []ServiceDefinition) error
	GetService(name string) (ServiceDefinition, error)
	UpdateService(name, display, audience, redirect string) error
	DeleteService(name string) (deleted bool, err error)
	ListServices() ([]ServiceDefinition, error)
}
