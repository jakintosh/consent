package service

import (
	"errors"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountNotFound    = errors.New("account not found")
	ErrServiceNotFound    = errors.New("service not found")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenNotFound      = errors.New("token not found")
	ErrInternal           = errors.New("internal error")
)

type Service struct {
	identityStore  IdentityStore
	refreshStore   RefreshStore
	catalog        *ServiceCatalog
	tokenIssuer    tokens.Issuer
	tokenValidator tokens.Validator
}

func New(
	identityStore IdentityStore,
	refreshStore RefreshStore,
	catalogDir string,
	issuer tokens.Issuer,
	validator tokens.Validator,
) *Service {
	return &Service{
		identityStore:  identityStore,
		refreshStore:   refreshStore,
		catalog:        NewServiceCatalog(catalogDir),
		tokenIssuer:    issuer,
		tokenValidator: validator,
	}
}

func (s *Service) Catalog() *ServiceCatalog {
	return s.catalog
}
