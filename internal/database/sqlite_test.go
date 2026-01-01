package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/database"
)

func setupStore(t *testing.T) *database.SQLiteStore {
	t.Helper()
	store := database.NewSQLiteStore(":memory:")
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestNewSQLiteStore_InMemory(t *testing.T) {
	t.Parallel()
	store := setupStore(t)

	// in-memory store is created successfully
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewSQLiteStore_CreatesSchema(t *testing.T) {
	t.Parallel()
	store := setupStore(t)

	// schema is created - insert and retrieve works
	err := store.InsertIdentity("test-user", []byte("secret-hash"))
	if err != nil {
		t.Fatalf("schema not created - InsertIdentity failed: %v", err)
	}

	secret, err := store.GetSecret("test-user")
	if err != nil {
		t.Fatalf("schema not created - GetSecret failed: %v", err)
	}
	if string(secret) != "secret-hash" {
		t.Errorf("unexpected secret: %s", string(secret))
	}
}

func TestSQLiteStore_Close(t *testing.T) {
	t.Parallel()
	store := database.NewSQLiteStore(":memory:")

	// closing store succeeds without error
	if err := store.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestSQLiteStore_IdentityStore(t *testing.T) {
	t.Parallel()
	store := setupStore(t)

	// IdentityStore returns the same store instance
	identityStore := store.IdentityStore()
	if identityStore == nil {
		t.Fatal("IdentityStore() returned nil")
	}
	if identityStore != store {
		t.Error("IdentityStore() should return the same store")
	}
}

func TestSQLiteStore_RefreshStore(t *testing.T) {
	t.Parallel()
	store := setupStore(t)

	// RefreshStore returns the same store instance
	refreshStore := store.RefreshStore()
	if refreshStore == nil {
		t.Fatal("RefreshStore() returned nil")
	}
	if refreshStore != store {
		t.Error("RefreshStore() should return the same store")
	}
}
