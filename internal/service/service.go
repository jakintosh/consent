// Package service implements the business logic layer for the consent identity server.
// It handles user authentication, registration, token management, and service management operations.
package service

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
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

// Options configures Service initialization.
type Options struct {
	Store                   Store
	KeysStore               keys.Store
	TokenServerOpts         tokens.ServerOptions
	ResourceTokenClientOpts tokens.ClientOptions
	PasswordMode            PasswordMode
}

// InitOptions configures bootstrap initialization for service state.
type InitOptions struct {
	Store          Store
	KeysStore      keys.Store
	PublicURL      string
	BootstrapToken string
}

// Service coordinates authentication, registration, and token operations.
// It depends on a Store interface and delegates to it for persistence.
type Service struct {
	store                  Store
	passwordMode           PasswordMode
	tokenIssuer            tokens.Issuer
	tokenValidator         tokens.Validator
	resourceTokenValidator tokens.Validator
	consentAPIAudience     string
	keys                   *keys.Service
}

func New(
	options Options,
) (
	*Service,
	error,
) {
	if options.Store == nil {
		return nil, errors.New("service: store required")
	}

	if options.KeysStore == nil {
		return nil, errors.New("service: keys store required")
	}

	keysSvc, err := NewKeysService(options.KeysStore)
	if err != nil {
		return nil, err
	}

	issuer, validator := tokens.InitServer(options.TokenServerOpts)
	resourceValidator := tokens.InitClient(options.ResourceTokenClientOpts)

	return &Service{
		passwordMode:           options.PasswordMode,
		store:                  options.Store,
		keys:                   keysSvc,
		tokenIssuer:            issuer,
		tokenValidator:         validator,
		resourceTokenValidator: resourceValidator,
		consentAPIAudience:     options.ResourceTokenClientOpts.ValidAudience,
	}, nil
}

func Init(
	options InitOptions,
) error {
	if err := EnsureSystemServices(options.Store, options.PublicURL); err != nil {
		return err
	}

	if err := InitKeys(options.KeysStore, options.BootstrapToken); err != nil {
		return err
	}

	return nil
}

func (s *Service) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /me", s.handleMe)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("POST /refresh", s.handleRefresh)
	mux.HandleFunc("POST /register", s.handleRegister)

	mux.HandleFunc("GET /services", s.keys.WithAuthFunc(s.handleListServices, &PermissionRead))
	mux.HandleFunc("POST /services", s.keys.WithAuthFunc(s.handleCreateService, &PermissionWrite))
	mux.HandleFunc("GET /services/{name}", s.keys.WithAuthFunc(s.handleGetService, &PermissionRead))
	mux.HandleFunc("PUT /services/{name}", s.keys.WithAuthFunc(s.handleUpdateService, &PermissionWrite))
	mux.HandleFunc("DELETE /services/{name}", s.keys.WithAuthFunc(s.handleDeleteService, &PermissionWrite))

	admin := http.NewServeMux()
	wire.Subrouter(admin, "/keys", s.keys.WithAuth(s.keys.Handler(), &PermissionAdmin))
	wire.Subrouter(mux, "/admin", admin)

	return mux
}

func decodeRequest[T any](r *http.Request) (T, error) {
	var req T
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}
