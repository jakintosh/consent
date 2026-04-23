package service

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type IdentityRecord struct {
	Subject string
	Handle  string
	Secret  []byte
	Roles   []string
}

type RoleDefinition struct {
	Name    string
	Display string
}

// Store handles persistence of identity data, refresh tokens, and services.
type Store interface {
	InsertUser(subject, handle string, secret []byte, roles []string) error
	GetUserByHandle(handle string) (IdentityRecord, error)
	GetUserBySubject(subject string) (IdentityRecord, error)
	ListUsers() ([]IdentityRecord, error)
	UpdateUser(subject, handle string, roles []string) error
	DeleteUser(subject string) (deleted bool, err error)

	InsertRole(name, display string) error
	GetRole(name string) (RoleDefinition, error)
	UpdateRoleDisplay(name, display string) error
	DeleteRole(name string) (deleted bool, err error)
	ListRoles() ([]RoleDefinition, error)
	CountUsersWithRole(name string) (int, error)
	ValidateRoleNames(names []string) error

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
