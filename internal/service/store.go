package service

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// Store handles persistence of user data, refresh tokens, and integrations.
type Store interface {
	InsertUser(subject string, handle string, secret []byte, roles []string) error
	GetUserByHandle(handle string) (*User, error)
	GetUserBySubject(subject string) (*User, error)
	ListUsers() ([]User, error)
	UpdateUser(subject string, handle string, roles []string) error
	DeleteUser(subject string) (deleted bool, err error)
	GetSecret(handle string) ([]byte, error)

	InsertRole(name string, display string) error
	GetRole(name string) (Role, error)
	UpdateRole(name string, updates *RoleUpdate) error
	DeleteRole(name string) (deleted bool, err error)
	ListRoles() ([]Role, error)

	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (subject string, err error)

	ListGrantedScopeNames(subject string, integration string) ([]string, error)
	InsertGrants(subject string, integration string, scopes []string) error

	InsertIntegration(name string, display string, audience string, redirect string, homepage string, logo string, requiredRoles []string) error
	GetIntegrationRoles(name string) ([]string, error)
	UpsertSystemIntegrations(integrations []Integration) error
	GetIntegration(name string) (Integration, error)
	GetIntegrationByAudience(audience string) (Integration, error)
	UpdateIntegration(name string, updates *IntegrationUpdate) error
	DeleteIntegration(name string) (deleted bool, err error)
	ListIntegrations() ([]Integration, error)
}
