// Package testutil provides test environment setup and utilities for internal package tests.
package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var (
	sharedSigningKey     *ecdsa.PrivateKey
	sharedSigningKeyOnce sync.Once
)

// getSharedSigningKey returns a cached ECDSA signing key for tests.
// This avoids the overhead of generating a new key for each test.
func getSharedSigningKey() *ecdsa.PrivateKey {
	sharedSigningKeyOnce.Do(func() {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic("failed to generate shared signing key: " + err.Error())
		}
		sharedSigningKey = key
	})
	return sharedSigningKey
}

// TestEnv provides all dependencies needed for testing
type TestEnv struct {
	DB             *database.SQLStore
	Service        *service.Service
	Router         http.Handler
	TokenIssuer    tokens.Issuer
	TokenValidator tokens.Validator
}

// SetupTestDB creates an in-memory SQLite database with cleanup.
func SetupTestDB(t *testing.T) *database.SQLStore {
	t.Helper()
	db, err := database.NewSQLStore(database.SQLStoreOptions{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to setup test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// TestUser provides credentials for seeding users in tests.
type TestUser struct {
	Handle   string
	Password string
}

// SetupTestEnv creates an isolated test environment with in-memory SQLite.
func SetupTestEnv(
	t *testing.T,
) *TestEnv {
	t.Helper()

	db := SetupTestDB(t)

	// use cached signing key (generated once across all tests)
	tkServerOpts := tokens.ServerOptions{
		SigningKey:   getSharedSigningKey(),
		IssuerDomain: "test.consent.local",
	}

	// create token issuer/validator for test helpers
	issuer, validator := tokens.InitServer(tkServerOpts)

	// create service
	serviceOpts := service.ServiceOptions{
		PasswordMode:    service.PasswordModeTesting,
		Store:           db,
		TokenServerOpts: tkServerOpts,
	}
	svc, err := service.New(serviceOpts)
	if err != nil {
		t.Fatalf("failed to initialize test service: %v", err)
	}

	return &TestEnv{
		DB:             db,
		Service:        svc,
		TokenIssuer:    issuer,
		TokenValidator: validator,
	}
}

// CreateTestService seeds a service definition for tests.
func (env *TestEnv) CreateTestService(
	t *testing.T,
	name string,
	display string,
	audience string,
	redirect string,
) {
	t.Helper()
	if err := env.Service.CreateService(name, display, audience, redirect); err != nil {
		t.Fatalf("failed to create test service: %v", err)
	}
}

// SetupTestEnvWithUsers creates TestEnv and registers the provided users.
func SetupTestEnvWithUsers(
	t *testing.T,
	users ...TestUser,
) *TestEnv {
	t.Helper()
	env := SetupTestEnv(t)
	for _, user := range users {
		env.RegisterTestUser(t, user.Handle, user.Password)
	}
	return env
}

// SetupTestEnvWithRouter creates TestEnv and configures the API router.
func SetupTestEnvWithRouter(
	t *testing.T,
) *TestEnv {
	t.Helper()
	env := SetupTestEnv(t)
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "http://localhost:8080/callback")
	env.Router = env.Service.Router()
	return env
}

// RegisterTestUser creates a test user in the database
func (env *TestEnv) RegisterTestUser(
	t *testing.T,
	handle string,
	password string,
) {
	t.Helper()
	if err := env.Service.Register(handle, password); err != nil {
		t.Fatalf("failed to register test user: %v", err)
	}
}

// IssueTestRefreshToken creates a refresh token for testing
func (env *TestEnv) IssueTestRefreshToken(
	t *testing.T,
	subject string,
	audience []string,
) *tokens.RefreshToken {
	t.Helper()
	token, err := env.TokenIssuer.IssueRefreshToken(subject, audience, time.Hour)
	if err != nil {
		t.Fatalf("failed to issue test refresh token: %v", err)
	}
	return token
}

// IssueTestAccessToken creates an access token for testing
func (env *TestEnv) IssueTestAccessToken(
	t *testing.T,
	subject string,
	audience []string,
) *tokens.AccessToken {
	t.Helper()
	token, err := env.TokenIssuer.IssueAccessToken(subject, audience, 30*time.Minute)
	if err != nil {
		t.Fatalf("failed to issue test access token: %v", err)
	}
	return token
}

// StoreTestRefreshToken issues and stores a refresh token in the database
func (env *TestEnv) StoreTestRefreshToken(
	t *testing.T,
	handle string,
	audience []string,
) *tokens.RefreshToken {
	t.Helper()
	token := env.IssueTestRefreshToken(t, handle, audience)
	if err := env.DB.InsertRefreshToken(token); err != nil {
		t.Fatalf("failed to store test refresh token: %v", err)
	}
	return token
}
