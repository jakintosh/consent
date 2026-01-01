package service_test

import (
	"errors"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login returns redirect URL with auth_code
	redirectURL, err := env.Service.Login("alice", "password123", "test-service")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL == nil {
		t.Fatal("expected redirect URL")
	}
	authCode := redirectURL.Query().Get("auth_code")
	if authCode == "" {
		t.Error("redirect URL missing auth_code parameter")
	}
}

func TestLogin_RedirectURL(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login redirects to service's configured callback URL
	redirectURL, err := env.Service.Login("alice", "password123", "test-service")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL.Host != "localhost:8080" {
		t.Errorf("redirect host = %s, want localhost:8080", redirectURL.Host)
	}
	if redirectURL.Path != "/callback" {
		t.Errorf("redirect path = %s, want /callback", redirectURL.Path)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns ErrInvalidCredentials
	_, err := env.Service.Login("alice", "wrongpassword", "test-service")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// unknown user returns ErrAccountNotFound
	_, err := env.Service.Login("unknown", "password", "test-service")
	if !errors.Is(err, service.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestLogin_UnknownService(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// unknown service returns ErrServiceNotFound
	_, err := env.Service.Login("alice", "password123", "nonexistent-service")
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Errorf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestLogin_StoresRefreshToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login and get auth_code
	redirectURL, err := env.Service.Login("alice", "password123", "test-service")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// token is stored and owner can be retrieved
	owner, err := env.DB.GetRefreshTokenOwner(authCode)
	if err != nil {
		t.Fatalf("Token not stored: %v", err)
	}
	if owner != "alice" {
		t.Errorf("token owner = %s, want alice", owner)
	}
}

func TestLogin_AuthCodeIsValidJWT(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login and get auth_code
	redirectURL, err := env.Service.Login("alice", "password123", "test-service")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// auth_code is valid JWT format (3 dot-separated parts)
	parts := strings.Split(authCode, ".")
	if len(parts) != 3 {
		t.Errorf("auth_code not valid JWT format, has %d parts", len(parts))
	}
}
