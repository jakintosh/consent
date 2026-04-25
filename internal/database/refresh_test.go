package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

var (
	testAudience1 = []string{"test-audience-1"}
	testAudience2 = []string{"test-audience-2"}
)

func TestInsertRefreshToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

	// setup env
	token := env.IssueTestRefreshToken(t, "alice", testAudience1)

	// inserting a refresh token succeeds
	err := store.InsertRefreshToken(token)
	if err != nil {
		t.Fatalf("InsertRefreshToken failed: %v", err)
	}
}

func TestInsertRefreshToken_NonExistentUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

	// insert token for non-existent user should succeed (subquery returns no rows)
	token := env.IssueTestRefreshToken(t, "nonexistent-user", testAudience1)
	err := store.InsertRefreshToken(token)
	if err != nil {
		t.Fatalf("InsertRefreshToken should succeed (no rows inserted): %v", err)
	}
}

func TestInsertRefreshToken_MultipleTokens(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	user, err := store.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if owner != user.Subject {
		t.Errorf("owner = %s, want %s", owner, user.Subject)
	}
}

func TestGetRefreshTokenOwner_NotExists(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

	// querying non-existent token returns error
	_, err := store.GetRefreshTokenOwner("nonexistent-jwt")
	if err == nil {
		t.Error("expected error for non-existent token")
	}
}

func TestGetRefreshTokenOwner_AfterDelete(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithUsers(t, testutil.TestUser{Handle: "alice", Password: "password"})
	store := env.DB

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
	env := testutil.SetupTestEnvWithUsers(
		t,
		testutil.TestUser{Handle: "alice", Password: "password"},
		testutil.TestUser{Handle: "bob", Password: "password"},
	)
	store := env.DB

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
	aliceUser, err := store.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle alice failed: %v", err)
	}
	if aliceOwner != aliceUser.Subject {
		t.Errorf("alice owner = %s, want %s", aliceOwner, aliceUser.Subject)
	}

	bobOwner, err := store.GetRefreshTokenOwner(bobToken.Encoded())
	if err != nil {
		t.Fatalf("GetRefreshTokenOwner bob failed: %v", err)
	}
	bobUser, err := store.GetUserByHandle("bob")
	if err != nil {
		t.Fatalf("GetUserByHandle bob failed: %v", err)
	}
	if bobOwner != bobUser.Subject {
		t.Errorf("bob owner = %s, want %s", bobOwner, bobUser.Subject)
	}
}
