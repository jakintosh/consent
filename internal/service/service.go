package service

import (
	"database/sql"
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
	db             *sql.DB
	catalog        *ServiceCatalog
	tokenIssuer    tokens.Issuer
	tokenValidator tokens.Validator
}

func New(
	dbPath string,
	catalogDir string,
	issuer tokens.Issuer,
	validator tokens.Validator,
) *Service {
	return &Service{
		db:             initDatabase(dbPath),
		catalog:        NewServiceCatalog(catalogDir),
		tokenIssuer:    issuer,
		tokenValidator: validator,
	}
}

func (s *Service) Catalog() *ServiceCatalog {
	return s.catalog
}
