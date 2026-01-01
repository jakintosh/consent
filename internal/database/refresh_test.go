package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

var (
	testAudience1 = []string{"test-audience-1"}
	testAudience2 = []string{"test-audience-2"}
)

func setupRefreshStore(t *testing.T) (*database.SQLiteStore, *testutil.TestEnv) {
	t.Helper()
	env := testutil.SetupTestEnv(t)

	// refresh env test data
	env.RegisterTestUser(t, "alice", "password")
	env.RegisterTestUser(t, "bob", "password")

	return env.DB, env
}

func TestInsertRefreshToken_Success(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// inserting a refresh token succeeds
	err := store.InsertRefreshToken(token)
	if err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}
}

func TestInsertRefreshToken_MultipleTokens(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token1 := env.IssueTestRefreshToken(t, "alice", testAudience1)
	token2 := env.IssueTestRefreshToken(t, "alice", testAudience2)

	// multiple tokens for same user can be stored
	if err := store.InsertRefreshToken(token1); err != nil {
		t.Fatalf("InsertRefreshToken token1 failed: %v", err)
	}
	if err := store.InsertRefreshToken(token2); err != nil {
		t.Fatalf("InsertRefreshToken token2 failed: %v", err)
	}
}

func TestDeleteRefreshToken_Exists(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// first insert the token
	if err := store.InsertRefreshToken(token); err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}

	// deleting existing token returns true
	deleted, err := store.DeleteRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("DeleteRefreshToken failed: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}
}

func TestDeleteRefreshToken_NotExists(t *testing.T) {
	t.Parallel()
	store, _ := setupRefreshStore(t)

	// deleting non-existent token returns false
	deleted, err := store.DeleteRefreshToken("nonexistent-jwt")
	if err != nil {
		t.Fatalf("DeleteRefreshToken failed: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for non-existent token")
	}
}

func TestDeleteRefreshToken_DoubleDelete(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// first insert the token
	if err := store.InsertRefreshToken(token); err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}

	// first delete succeeds
	deleted1, err := store.DeleteRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("DeleteRefreshToken first call failed: %v", err)
	}
	if !deleted1 {
		t.Error("expected first delete to return true")
	}

	// second delete fails
	deleted2, err := store.DeleteRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("DeleteRefreshToken second call failed: %v", err)
	}
	if deleted2 {
		t.Error("expected second delete to return false")
	}
}

func TestGetRefreshTokenOwner_Exists(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// first insert the token
	if err := store.InsertRefreshToken(token); err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}

	// owner is returned for existing token
	owner, err := store.GetRefreshTokenOwner(token.Encoded())
	if err != nil {
		t.Fatalf("GetRefreshTokenOwner failed: %v", err)
	}
	if owner != "alice" {
		t.Errorf("owner = %s, want alice", owner)
	}
}

func TestGetRefreshTokenOwner_NotExists(t *testing.T) {
	t.Parallel()
	store, _ := setupRefreshStore(t)

	// querying non-existent token returns error
	_, err := store.GetRefreshTokenOwner("nonexistent-jwt")
	if err == nil {
		t.Error("expected error for non-existent token")
	}
}

func TestGetRefreshTokenOwner_AfterDelete(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// first insert the token
	if err := store.InsertRefreshToken(token); err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}

	// delete the token
	_, _ = store.DeleteRefreshToken(token.Encoded())

	// querying deleted token returns error
	_, err := store.GetRefreshTokenOwner(token.Encoded())
	if err == nil {
		t.Error("expected error for deleted token")
	}
}

func TestRefreshToken_MultipleUsers(t *testing.T) {
	t.Parallel()
	store, env := setupRefreshStore(t)

	// setup env
	aliceToken := env.IssueTestRefreshToken(t, "alice", testAudience1)
	bobToken := env.IssueTestRefreshToken(t, "bob", testAudience1)

	// store tokens for both users
	if err := store.InsertRefreshToken(aliceToken); err != nil {
		t.Fatalf("InsertRefreshToken alice failed: %v", err)
	}
	if err := store.InsertRefreshToken(bobToken); err != nil {
		t.Fatalf("InsertRefreshToken bob failed: %v", err)
	}

	// each token returns correct owner
	aliceOwner, err := store.GetRefreshTokenOwner(aliceToken.Encoded())
	if err != nil {
		t.Fatalf("GetRefreshTokenOwner alice failed: %v", err)
	}
	if aliceOwner != "alice" {
		t.Errorf("alice owner = %s, want alice", aliceOwner)
	}

	bobOwner, err := store.GetRefreshTokenOwner(bobToken.Encoded())
	if err != nil {
		t.Fatalf("GetRefreshTokenOwner bob failed: %v", err)
	}
	if bobOwner != "bob" {
		t.Errorf("bob owner = %s, want bob", bobOwner)
	}
}
