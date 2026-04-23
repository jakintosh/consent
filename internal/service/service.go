// Package service implements the business logic layer for the consent identity server.
// It handles user authentication, registration, token management, and service management operations.
package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"golang.org/x/crypto/bcrypt"
)

var (
	PermissionRead = keys.Permission{
		Key:         "read",
		Display:     "Read",
		Description: "Read-only API access",
	}
	PermissionWrite = keys.Permission{
		Key:         "write",
		Display:     "Write",
		Description: "Mutating API access",
	}
	PermissionAdmin = keys.Permission{
		Key:         "admin",
		Display:     "Admin",
		Description: "Administrative access",
	}
)

func AllKeyPermissions() []keys.Permission {
	return []keys.Permission{
		PermissionRead,
		PermissionWrite,
		PermissionAdmin,
	}
}

func AllKeyPermissionRefs() []*keys.Permission {
	return []*keys.Permission{
		&PermissionRead,
		&PermissionWrite,
		&PermissionAdmin,
	}
}

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

	issuer, validator := tokens.InitServer(options.TokenServerOpts)
	resourceValidator := tokens.InitClient(options.ResourceTokenClientOpts)

	return &Service{
		passwordMode:           options.PasswordMode,
		store:                  options.Store,
		tokenIssuer:            issuer,
		tokenValidator:         validator,
		resourceTokenValidator: resourceValidator,
		consentAPIAudience:     options.ResourceTokenClientOpts.ValidAudience,
	}, nil
}

func Init(
	options InitOptions,
) error {
	if err := SeedSystemServices(options.Store, options.PublicURL); err != nil {
		return err
	}

	if err := SeedSystemRoles(options.Store); err != nil {
		return err
	}

	if options.BootstrapToken == "" {
		return fmt.Errorf("service: bootstrap token required")
	}
	if options.KeysStore == nil {
		return fmt.Errorf("service: keys store required")
	}

	opts := keys.Options{
		Store:       options.KeysStore,
		Permissions: AllKeyPermissions(),
	}
	keysSvc, err := keys.New(opts)
	if err != nil {
		return err
	}

	err = keysSvc.Init(options.BootstrapToken, AllKeyPermissionRefs()...)
	if err != nil {
		if !errors.Is(err, keys.ErrAlreadyInitialized) {
			return fmt.Errorf("service: initialize keys: %w", err)
		}
	}

	return nil
}

func generateSubject() (
	string,
	error,
) {
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate subject: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func parseAndValidateRedirectURL(
	redirect string,
) (
	*url.URL,
	error,
) {
	parsed, err := url.Parse(redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}
	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidUrl
	}
	return parsed, nil
}
