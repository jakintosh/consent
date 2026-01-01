// Package testutil provides test environment setup and utilities for internal package tests.
package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/api"
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
	DB             *database.SQLiteStore
	Service        *service.Service
	Router         http.Handler
	TokenIssuer    tokens.Issuer
	TokenValidator tokens.Validator
}

// SetupTestEnv creates an isolated test environment with in-memory SQLite
func SetupTestEnv(
	t *testing.T,
) *TestEnv {
	t.Helper()

	// create in-memory SQLite database
	db := database.NewSQLiteStore(":memory:")

	// use cached signing key (generated once across all tests)
	signingKey := getSharedSigningKey()

	// create token issuer/validator
	issuer, validator := tokens.InitServer(signingKey, "test.consent.local")

	// get path to testdata/services
	servicesDir := getTestDataPath("services")

	// create service
	svc := service.New(
		db.IdentityStore(),
		db.RefreshStore(),
		servicesDir,
		issuer,
		validator,
		service.PasswordModeTesting,
	)

	// setup cleanup
	t.Cleanup(func() {
		_ = db.Close()
	})

	return &TestEnv{
		DB:             db,
		Service:        svc,
		TokenIssuer:    issuer,
		TokenValidator: validator,
	}
}

// SetupTestEnvWithRouter creates TestEnv and configures the API router
func SetupTestEnvWithRouter(
	t *testing.T,
) *TestEnv {
	t.Helper()
	env := SetupTestEnv(t)
	a := api.New(env.Service)
	env.Router = a.Router()
	return env
}

// getTestDataPath returns the path to a subdirectory in testdata
func getTestDataPath(
	subdir string,
) string {
	_, filename, _, _ := runtime.Caller(0)
	// Go up from internal/testutil to repo root, then into testdata
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", subdir)
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
