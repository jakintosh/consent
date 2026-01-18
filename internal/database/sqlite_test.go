package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestNewSQLiteStore_InMemory(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// in-memory store is created successfully
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewSQLiteStore_CreatesSchema(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

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
	// Close is called explicitly to validate success without test cleanup.
	store := database.NewSQLiteStore(database.SQLStoreOptions{Path: ":memory:"})

	// closing store succeeds without error
	if err := store.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
