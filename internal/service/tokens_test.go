package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestRefreshTokens_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refreshing valid token returns new access and refresh tokens
	accessToken, newRefreshToken, err := env.Service.RefreshTokens(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshTokens failed: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if newRefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// malformed token returns ErrTokenInvalid
	_, _, err := env.Service.RefreshTokens("invalid-token")
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestRefreshTokens_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns ErrTokenNotFound
	_, _, err := env.Service.RefreshTokens(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRefreshTokens_DeletesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, _, err := env.Service.RefreshTokens(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshTokens failed: %v", err)
	}

	// old token is deleted and can't be used again
	_, _, err = env.Service.RefreshTokens(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("old token should be deleted, got %v", err)
	}
}

func TestRefreshTokens_StoresNewToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refresh returns new token
	_, newRefreshToken, err := env.Service.RefreshTokens(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshTokens failed: %v", err)
	}

	// new token is stored in database
	owner, err := env.DB.GetRefreshTokenOwner(newRefreshToken)
	if err != nil {
		t.Fatalf("new token not stored: %v", err)
	}
	if owner != "alice" {
		t.Errorf("new token owner = %s, want alice", owner)
	}
}

func TestRefreshTokens_CanBeRefreshedAgain(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, newRefreshToken1, err := env.Service.RefreshTokens(token.Encoded())
	if err != nil {
		t.Fatalf("First RefreshTokens failed: %v", err)
	}

	// new token can be used for another refresh
	_, newRefreshToken2, err := env.Service.RefreshTokens(newRefreshToken1)
	if err != nil {
		t.Fatalf("Second RefreshTokens failed: %v", err)
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
	_, _, err := env.Service.RefreshTokens(token.Encoded())
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
