// Package service implements the business logic layer for the consent identity server.
// It handles user authentication, registration, token management, and service catalog operations.
package service

import (
	"errors"
	"log"
	"os"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"golang.org/x/crypto/bcrypt"
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

// PasswordMode controls bcrypt cost for password hashing.
// Use PasswordModeProduction for real deployments and PasswordModeTesting only in tests.
type PasswordMode int

const (
	// PasswordModeProduction uses bcrypt.DefaultCost (10) for secure password hashing.
	PasswordModeProduction PasswordMode = iota
	// PasswordModeTesting uses bcrypt.MinCost (4) for fast test execution.
	// WARNING: This mode will panic if used outside of go test.
	PasswordModeTesting
)

// Cost returns the bcrypt cost for this mode.
// Panics if PasswordModeTesting is used outside of a test environment.
func (m PasswordMode) Cost() int {
	switch m {
	case PasswordModeTesting:
		// Safety check: only allow testing mode during go test
		// Go sets this environment variable automatically when running tests
		if os.Getenv("GO_TEST_TIMEOUT_SCALE") == "" && os.Getenv("GO_TEST") == "" {
			// Check if running under go test by looking for test flags
			for _, arg := range os.Args {
				if arg == "-test.v" || arg == "-test.run" || len(arg) > 5 && arg[:6] == "-test." {
					goto allowed
				}
			}
			panic("service: PasswordModeTesting used outside of test environment")
		}
	allowed:
		log.Println("WARNING: Using insecure password hashing (testing mode)")
		return bcrypt.MinCost
	default:
		return bcrypt.DefaultCost
	}
}

// Service coordinates authentication, registration, and token operations.
// It depends on storage interfaces (IdentityStore, RefreshStore) and delegates
// to them for persistence.
type Service struct {
	identityStore  IdentityStore
	refreshStore   RefreshStore
	catalog        *ServiceCatalog
	tokenIssuer    tokens.Issuer
	tokenValidator tokens.Validator
	passwordMode   PasswordMode
}

func New(
	identityStore IdentityStore,
	refreshStore RefreshStore,
	catalogDir string,
	issuer tokens.Issuer,
	validator tokens.Validator,
	passwordMode PasswordMode,
) *Service {
	return &Service{
		identityStore:  identityStore,
		refreshStore:   refreshStore,
		catalog:        NewServiceCatalog(catalogDir),
		tokenIssuer:    issuer,
		tokenValidator: validator,
		passwordMode:   passwordMode,
	}
}

func (s *Service) Catalog() *ServiceCatalog {
	return s.catalog
}
