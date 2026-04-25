package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestListGrantedScopeNames_Empty(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	scopes, err := store.ListGrantedScopeNames("subject-alice", "nonexistent")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 0 {
		t.Errorf("len(scopes) = %d, want 0", len(scopes))
	}
}

func TestListGrantedScopeNames_WithGrants(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	err := store.InsertGrants("subject-alice", "test-integration", []string{"read", "write", "admin"})
	if err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	scopes, err := store.ListGrantedScopeNames("subject-alice", "test-integration")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 3 {
		t.Fatalf("len(scopes) = %d, want 3", len(scopes))
	}
	scopeSet := make(map[string]bool)
	for _, s := range scopes {
		scopeSet[s] = true
	}
	if !scopeSet["read"] || !scopeSet["write"] || !scopeSet["admin"] {
		t.Errorf("scopes = %#v, want read, write, admin", scopes)
	}
}

func TestListGrantedScopeNames_WrongIntegration(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	err := store.InsertGrants("subject-alice", "integration-a", []string{"read"})
	if err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	scopes, err := store.ListGrantedScopeNames("subject-alice", "integration-b")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 0 {
		t.Errorf("len(scopes) = %d, want 0", len(scopes))
	}
}

func TestInsertGrants_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	err := store.InsertGrants("subject-alice", "test-integration", []string{"read", "write"})
	if err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	scopes, err := store.ListGrantedScopeNames("subject-alice", "test-integration")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 2 {
		t.Fatalf("len(scopes) = %d, want 2", len(scopes))
	}
}

func TestInsertGrants_Idempotent(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	err := store.InsertGrants("subject-alice", "test-integration", []string{"read"})
	if err != nil {
		t.Fatalf("InsertGrants first call failed: %v", err)
	}

	err = store.InsertGrants("subject-alice", "test-integration", []string{"read"})
	if err != nil {
		t.Fatalf("InsertGrants second call failed: %v", err)
	}

	scopes, err := store.ListGrantedScopeNames("subject-alice", "test-integration")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 1 {
		t.Fatalf("len(scopes) = %d, want 1 (idempotent)", len(scopes))
	}
}

func TestInsertGrants_NonExistentUser(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// InsertGrants for non-existent user should not fail at the DB layer
	// (the FK constraint on owner prevents it, but the subquery returns no rows)
	err := store.InsertGrants("nonexistent-user", "test-integration", []string{"read"})
	if err != nil {
		t.Fatalf("InsertGrants should succeed for non-existent user (no rows inserted): %v", err)
	}

	scopes, err := store.ListGrantedScopeNames("nonexistent-user", "test-integration")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	if len(scopes) != 0 {
		t.Fatalf("len(scopes) = %d, want 0 (no grants for non-existent user)", len(scopes))
	}
}

func TestInsertGrants_EmptyScopes(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	// Empty scopes should be a no-op
	err := store.InsertGrants("subject-alice", "test-integration", []string{})
	if err != nil {
		t.Fatalf("InsertGrants with empty scopes failed: %v", err)
	}
}
