package service

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// Store handles persistence of user data, refresh tokens, and integrations.
type Store interface {
	InsertUser(subject, handle string, secret []byte, roles []string) error
	GetUserByHandle(handle string) (*User, error)
	GetUserBySubject(subject string) (*User, error)
	ListUsers() ([]User, error)
	UpdateUser(subject, handle string, roles []string) error
	DeleteUser(subject string) (deleted bool, err error)
	GetSecret(handle string) ([]byte, error)

	InsertRole(name, display string) error
	GetRole(name string) (Role, error)
	UpdateRole(name string, updates *RoleUpdate) error
	DeleteRole(name string) (deleted bool, err error)
	ListRoles() ([]Role, error)

	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (subject string, err error)

	ListGrantedScopeNames(subject, integration string) ([]string, error)
	InsertGrants(subject, integration string, scopes []string) error

	InsertIntegration(name, display, audience, redirect string) error
	UpsertSystemIntegrations(integrations []Integration) error
	GetIntegration(name string) (Integration, error)
	UpdateIntegration(name string, updates *IntegrationUpdate) error
	DeleteIntegration(name string) (deleted bool, err error)
	ListIntegrations() ([]Integration, error)
}
