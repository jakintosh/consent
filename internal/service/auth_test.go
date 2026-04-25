package service_test

import (
	"errors"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestGrantAuthCode_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// returns redirect URL with auth_code
	redirectURL, err := env.Service.GrantAuthCode("alice", "password123", service.InternalIntegrationName)
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

func TestGrantAuthCode_RedirectURL(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// redirects to the integration's configured callback URL
	redirectURL, err := env.Service.GrantAuthCode("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL.Host != "consent.test" {
		t.Errorf("redirect host = %s, want consent.test", redirectURL.Host)
	}
	if redirectURL.Path != "/auth/callback" {
		t.Errorf("redirect path = %s, want /auth/callback", redirectURL.Path)
	}
}

func TestGrantAuthCode_WrongPassword(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns ErrInvalidCredentials
	_, err := env.Service.GrantAuthCode("alice", "wrongpassword", service.InternalIntegrationName)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGrantAuthCode_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// unknown user returns ErrAccountNotFound
	_, err := env.Service.GrantAuthCode("unknown", "password", service.InternalIntegrationName)
	if !errors.Is(err, service.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestGrantAuthCode_UnknownIntegration(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// unknown integration returns ErrIntegrationNotFound
	_, err := env.Service.GrantAuthCode("alice", "password123", "nonexistent-service")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Errorf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestGrantAuthCode_StoresRefreshToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// grant auth_code and get redirect
	redirectURL, err := env.Service.GrantAuthCode("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// token is stored and owner can be retrieved
	owner, err := env.DB.GetRefreshTokenOwner(authCode)
	if err != nil {
		t.Fatalf("Token not stored: %v", err)
	}
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if owner != user.Subject {
		t.Errorf("token owner = %s, want %s", owner, user.Subject)
	}
}

func TestGrantAuthCode_AuthCodeIsValidJWT(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// grant auth_code and get redirect
	redirectURL, err := env.Service.GrantAuthCode("alice", "password123", service.InternalIntegrationName)
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

func TestRefreshAccessToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refreshing valid token returns new access and refresh tokens
	accessToken, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if newRefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestRefreshAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// malformed token returns ErrTokenInvalid
	_, _, err := env.Service.RefreshAccessToken("invalid-token")
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestRefreshAccessToken_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns ErrTokenNotFound
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRefreshAccessToken_DeletesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// old token is deleted and can't be used again
	_, _, err = env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("old token should be deleted, got %v", err)
	}
}

func TestRefreshAccessToken_StoresNewToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refresh returns new token
	_, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// new token is stored in database
	owner, err := env.DB.GetRefreshTokenOwner(newRefreshToken)
	if err != nil {
		t.Fatalf("new token not stored: %v", err)
	}
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if owner != user.Subject {
		t.Errorf("new token owner = %s, want %s", owner, user.Subject)
	}
}

func TestRefreshAccessToken_CanBeRefreshedAgain(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, newRefreshToken1, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("First RefreshAccessToken failed: %v", err)
	}

	// new token can be used for another refresh
	_, newRefreshToken2, err := env.Service.RefreshAccessToken(newRefreshToken1)
	if err != nil {
		t.Fatalf("Second RefreshAccessToken failed: %v", err)
	}
	if newRefreshToken2 == "" {
		t.Error("expected non-empty second refresh token")
	}
}

func TestRevokeRefreshToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoking valid token succeeds
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
}

func TestRevokeRefreshToken_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// revoking non-existent token returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken("nonexistent-token")
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRevokeRefreshToken_CantRefreshAfterRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoke the token
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}

	// revoked token can't be used for refresh
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeRefreshToken_DoubleRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first revoke succeeds
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("First RevokeRefreshToken failed: %v", err)
	}

	// second revoke returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound on second revoke, got %v", err)
	}
}
