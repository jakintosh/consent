// Package service implements the business logic layer for the consent identity server.
// It handles user authentication, registration, token management, and service management operations.
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"golang.org/x/crypto/bcrypt"
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

// ServiceOptions configures Service initialization.
type ServiceOptions struct {
	Store           Store
	TokenServerOpts tokens.ServerOptions
	PasswordMode    PasswordMode
	KeysOptions     keys.Options
	PublicURL       string
}

// Service coordinates authentication, registration, and token operations.
// It depends on a Store interface and delegates to it for persistence.
type Service struct {
	store          Store
	passwordMode   PasswordMode
	tokenIssuer    tokens.Issuer
	tokenValidator tokens.Validator
	keys           *keys.Service
}

func New(
	options ServiceOptions,
) (
	*Service,
	error,
) {
	if options.Store == nil {
		return nil, errors.New("service: store required")
	}

	if options.KeysOptions.Store == nil {
		return nil, errors.New("service: keys options store required")
	}

	keysSvc, err := keys.New(options.KeysOptions)
	if err != nil {
		return nil, err
	}

	issuer, validator := tokens.InitServer(options.TokenServerOpts)

	internalService, err := BuildInternalServiceDefinition(options.PublicURL)
	if err != nil {
		return nil, fmt.Errorf("service: failed to build internal service: %w", err)
	}

	systemServices := []ServiceDefinition{internalService}
	err = options.Store.UpsertSystemServices(systemServices)
	if err != nil {
		return nil, fmt.Errorf("service: failed to initialize system services: %v", err)
	}

	return &Service{
		passwordMode:   options.PasswordMode,
		store:          options.Store,
		keys:           keysSvc,
		tokenIssuer:    issuer,
		tokenValidator: validator,
	}, nil
}

func (s *Service) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("POST /refresh", s.handleRefresh)
	mux.HandleFunc("POST /register", s.handleRegister)
	mux.HandleFunc("GET /me", s.handleMe)

	auth := s.keys.WithAuth
	mux.HandleFunc("GET /services", auth(s.handleListServices))
	mux.HandleFunc("POST /services", auth(s.handleCreateService))
	mux.HandleFunc("GET /services/{name}", auth(s.handleGetService))
	mux.HandleFunc("PUT /services/{name}", auth(s.handleUpdateService))
	mux.HandleFunc("DELETE /services/{name}", auth(s.handleDeleteService))

	s.keys.Router(mux, "", auth)

	return mux
}

func decodeRequest[T any](r *http.Request) (T, error) {
	var req T
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}
